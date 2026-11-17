package translation_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/translation"
)

func TestMyMemoryClient_TranslateAndCache(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.URL.Query().Get("q"); got != "hello" {
			t.Fatalf("expected q=hello, got %q", got)
		}
		if got := r.URL.Query().Get("langpair"); got != "en|es" {
			t.Fatalf("expected langpair=en|es, got %q", got)
		}
		if got := r.URL.Query().Get("de"); got != "dev@wordtap.app" {
			t.Fatalf("expected de=dev@wordtap.app, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"responseStatus": 200,
			"responseData": {"translatedText": "hola", "match": 1},
			"matches": [
				{"id": "101", "segment": "hello", "translation": "hola", "match": 1},
				{"id": "102", "segment": "hello", "translation": "saludos", "match": 0.85}
			]
		}`))
	}))
	defer srv.Close()

	client := translation.NewMyMemoryClientWithConfig(srv.URL, "dev@wordtap.app", srv.Client())

	entries, err := client.Translate(context.Background(), "hello", "en|es")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 deduplicated entries, got %d", len(entries))
	}
	if entries[0].Translation != "hola" || entries[0].Quality != 100 {
		t.Fatalf("unexpected primary entry: %+v", entries[0])
	}
	if entries[1].Translation != "saludos" || entries[1].Quality != 85 {
		t.Fatalf("unexpected secondary entry: %+v", entries[1])
	}

	// Second call must hit in-memory cache without hitting HTTP server.
	cached, err := client.Translate(context.Background(), "hello", "en|es")
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if len(cached) != 2 || hits.Load() != 1 {
		t.Fatalf("expected cache hit (1 server request), got hits=%d", hits.Load())
	}
}

func TestMyMemoryClient_RejectsOver500Bytes(t *testing.T) {
	client := translation.NewMyMemoryClientWithConfig("http://127.0.0.1:0", "", nil)
	_, err := client.Translate(context.Background(), strings.Repeat("a", 501), "en|es")
	if err == nil {
		t.Fatal("expected error for >500 bytes query")
	}
}

func TestMyMemoryClient_LRUEvictsOldestWithoutFlushingCache(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"responseStatus":200,"responseData":{"translatedText":"tr-%s","match":1}}`, q)
	}))
	defer srv.Close()

	client := translation.NewMyMemoryClientWithCapacity(srv.URL, "", srv.Client(), 2)
	ctx := context.Background()

	if _, err := client.Translate(ctx, "one", "en|es"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Translate(ctx, "two", "en|es"); err != nil {
		t.Fatal(err)
	}
	// Promote "one" to most-recently-used so "two" is evicted when "three" arrives.
	if _, err := client.Translate(ctx, "one", "en|es"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Translate(ctx, "three", "en|es"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 3 {
		t.Fatalf("expected 3 upstream hits, got %d", hits.Load())
	}

	// "one" and "three" must remain cached; "two" was evicted and must refetch.
	if _, err := client.Translate(ctx, "one", "en|es"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Translate(ctx, "three", "en|es"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 3 {
		t.Fatalf("expected MRU entries to stay cached, got %d hits", hits.Load())
	}
	if _, err := client.Translate(ctx, "two", "en|es"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 4 {
		t.Fatalf("expected evicted entry 'two' to trigger 4th upstream request, got %d", hits.Load())
	}
}

func TestMyMemoryClient_DeduplicatesConcurrentRequests(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(40 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseStatus":200,"responseData":{"translatedText":"mundo","match":1}}`))
	}))
	defer srv.Close()

	client := translation.NewMyMemoryClientWithConfig(srv.URL, "", srv.Client())

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entries, err := client.Translate(context.Background(), "world", "en|es")
			if err != nil {
				errs <- err
				return
			}
			if len(entries) != 1 || entries[0].Translation != "mundo" {
				errs <- fmt.Errorf("unexpected entries: %+v", entries)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected singleflight deduplication (1 upstream hit), got %d", hits.Load())
	}
}
