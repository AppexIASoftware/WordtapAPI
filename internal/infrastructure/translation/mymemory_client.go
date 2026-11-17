package translation

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultMyMemoryURL = "https://api.mymemory.translated.net/get"
	maxQueryBytes      = 500
	defaultCacheTTL    = 24 * time.Hour
	maxCacheEntries    = 2000
	maxResponseBytes   = 256 * 1024
)

// Entry represents a single translation candidate returned by MyMemory.
type Entry struct {
	ID          string
	Source      string
	Translation string
	Quality     int
}

type cachedResult struct {
	key       string
	entries   []Entry
	expiresAt time.Time
}

type inflightCall struct {
	done    chan struct{}
	entries []Entry
	err     error
}

// MyMemoryClient queries MyMemory's public translation API with an LRU TTL cache
// and concurrent singleflight request deduplication.
type MyMemoryClient struct {
	baseURL    string
	email      string
	httpClient *http.Client
	cacheTTL   time.Duration
	maxEntries int

	mu      sync.Mutex
	cache   map[string]*list.Element
	lruList *list.List

	inMu     sync.Mutex
	inflight map[string]*inflightCall
}

// NewMyMemoryClient creates a MyMemoryClient using an optional contact email (increases daily quota).
func NewMyMemoryClient(email string) *MyMemoryClient {
	return NewMyMemoryClientWithConfig(defaultMyMemoryURL, email, &http.Client{Timeout: 5 * time.Second})
}

// NewMyMemoryClientWithConfig creates a MyMemoryClient with custom endpoint and HTTP client (for testing).
func NewMyMemoryClientWithConfig(baseURL, email string, httpClient *http.Client) *MyMemoryClient {
	return NewMyMemoryClientWithCapacity(baseURL, email, httpClient, maxCacheEntries)
}

// NewMyMemoryClientWithCapacity creates a MyMemoryClient with an explicit LRU entry limit.
func NewMyMemoryClientWithCapacity(baseURL, email string, httpClient *http.Client, maxEntries int) *MyMemoryClient {
	if baseURL == "" {
		baseURL = defaultMyMemoryURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	if maxEntries <= 0 {
		maxEntries = maxCacheEntries
	}
	return &MyMemoryClient{
		baseURL:    baseURL,
		email:      strings.TrimSpace(email),
		httpClient: httpClient,
		cacheTTL:   defaultCacheTTL,
		maxEntries: maxEntries,
		cache:      make(map[string]*list.Element),
		lruList:    list.New(),
		inflight:   make(map[string]*inflightCall),
	}
}

type myMemoryAPIResponse struct {
	ResponseStatus  int    `json:"responseStatus"`
	ResponseDetails string `json:"responseDetails"`
	ResponseData    struct {
		TranslatedText string  `json:"translatedText"`
		Match          float64 `json:"match"`
	} `json:"responseData"`
	Matches []struct {
		ID          json.RawMessage `json:"id"`
		Segment     string          `json:"segment"`
		Translation string          `json:"translation"`
		Match       float64         `json:"match"`
	} `json:"matches"`
}

// Translate fetches translations for term using the given MyMemory langPair (e.g. "en|es").
func (c *MyMemoryClient) Translate(ctx context.Context, term, langPair string) ([]Entry, error) {
	cleanTerm := strings.TrimSpace(term)
	if cleanTerm == "" {
		return []Entry{}, nil
	}
	if len([]byte(cleanTerm)) > maxQueryBytes {
		return nil, errors.New("el texto a traducir supera el límite de 500 bytes")
	}

	cacheKey := strings.ToLower(langPair + "::" + cleanTerm)
	if entries, ok := c.getFromCache(cacheKey); ok {
		return entries, nil
	}

	c.inMu.Lock()
	if call, ok := c.inflight[cacheKey]; ok {
		c.inMu.Unlock()
		select {
		case <-call.done:
			return cloneEntries(call.entries), call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &inflightCall{done: make(chan struct{})}
	c.inflight[cacheKey] = call
	c.inMu.Unlock()

	go func() {
		reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		entries, err := c.fetchAndCache(reqCtx, cacheKey, cleanTerm, langPair)
		call.entries = entries
		call.err = err

		c.inMu.Lock()
		delete(c.inflight, cacheKey)
		c.inMu.Unlock()
		close(call.done)
	}()

	select {
	case <-call.done:
		return cloneEntries(call.entries), call.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *MyMemoryClient) fetchAndCache(ctx context.Context, cacheKey, cleanTerm, langPair string) ([]Entry, error) {
	reqURL, err := c.buildURL(cleanTerm, langPair)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error consultando MyMemory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MyMemory respondió con estado %d", resp.StatusCode)
	}

	var payload myMemoryAPIResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("error decodificando respuesta de MyMemory: %w", err)
	}
	if payload.ResponseStatus != 200 && payload.ResponseStatus != 0 {
		return nil, fmt.Errorf("MyMemory error %d: %s", payload.ResponseStatus, payload.ResponseDetails)
	}

	entries := extractEntries(cleanTerm, payload)
	c.putInCache(cacheKey, entries)
	return entries, nil
}

func (c *MyMemoryClient) buildURL(term, langPair string) (string, error) {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	q := parsed.Query()
	q.Set("q", term)
	q.Set("langpair", langPair)
	if c.email != "" {
		q.Set("de", c.email)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func extractEntries(sourceTerm string, payload myMemoryAPIResponse) []Entry {
	seen := make(map[string]struct{})
	entries := make([]Entry, 0, len(payload.Matches)+1)

	add := func(id, src, tr string, match float64) {
		cleanTr := strings.TrimSpace(tr)
		if cleanTr == "" {
			return
		}
		key := strings.ToLower(cleanTr)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}

		cleanSrc := strings.TrimSpace(src)
		if cleanSrc == "" {
			cleanSrc = sourceTerm
		}
		quality := int(match * 100)
		if quality < 0 {
			quality = 0
		} else if quality > 100 {
			quality = 100
		}
		entries = append(entries, Entry{
			ID:          id,
			Source:      cleanSrc,
			Translation: cleanTr,
			Quality:     quality,
		})
	}

	if payload.ResponseData.TranslatedText != "" {
		add("primary", sourceTerm, payload.ResponseData.TranslatedText, payload.ResponseData.Match)
	}

	for idx, m := range payload.Matches {
		id := strings.Trim(string(m.ID), `"`)
		if id == "" || id == "0" {
			id = fmt.Sprintf("match-%d", idx)
		}
		add(id, m.Segment, m.Translation, m.Match)
	}

	return entries
}

func (c *MyMemoryClient) getFromCache(key string) ([]Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	item := elem.Value.(*cachedResult)
	if time.Now().After(item.expiresAt) {
		c.lruList.Remove(elem)
		delete(c.cache, key)
		return nil, false
	}
	c.lruList.MoveToFront(elem)
	return cloneEntries(item.entries), true
}

func (c *MyMemoryClient) putInCache(key string, entries []Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cloned := cloneEntries(entries)
	expiresAt := time.Now().Add(c.cacheTTL)

	if elem, ok := c.cache[key]; ok {
		item := elem.Value.(*cachedResult)
		item.entries = cloned
		item.expiresAt = expiresAt
		c.lruList.MoveToFront(elem)
		return
	}

	if c.lruList.Len() >= c.maxEntries {
		now := time.Now()
		for k, elem := range c.cache {
			if now.After(elem.Value.(*cachedResult).expiresAt) {
				c.lruList.Remove(elem)
				delete(c.cache, k)
			}
		}
		for c.lruList.Len() >= c.maxEntries {
			oldest := c.lruList.Back()
			if oldest == nil {
				break
			}
			c.lruList.Remove(oldest)
			delete(c.cache, oldest.Value.(*cachedResult).key)
		}
	}

	elem := c.lruList.PushFront(&cachedResult{
		key:       key,
		entries:   cloned,
		expiresAt: expiresAt,
	})
	c.cache[key] = elem
}

func cloneEntries(entries []Entry) []Entry {
	if entries == nil {
		return nil
	}
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}
