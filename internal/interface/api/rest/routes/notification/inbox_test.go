package notification

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/labstack/echo/v5"
)

type inboxFake struct {
	user  string
	limit int
	found bool
	err   error
	calls int
}

func (f *inboxFake) List(_ context.Context, user string, limit int, _ *inbox.Cursor) (inbox.Page, error) {
	f.user = user
	f.limit = limit
	f.calls++
	return inbox.Page{}, f.err
}
func (f *inboxFake) UnreadCount(_ context.Context, user string) (int64, error) {
	f.user = user
	f.calls++
	return 2, f.err
}
func (f *inboxFake) MarkRead(_ context.Context, user, _ string) (bool, error) {
	f.user = user
	f.calls++
	return f.found, f.err
}
func (f *inboxFake) Create(_ context.Context, _ *entities.Notification) error {
	f.calls++
	return f.err
}
func (f *inboxFake) CreateBatch(_ context.Context, _ []entities.Notification) error {
	f.calls++
	return f.err
}

func TestInboxRoutes(t *testing.T) {
	id := "b2a3a528-4591-4bee-bfbb-88e06965d87c"
	for _, tt := range []struct {
		name, method, path, user string
		found                    bool
		err                      error
		status, calls            int
	}{
		{"unauthorized", "GET", "/notifications", "", false, nil, 401, 0},
		{"list", "GET", "/notifications?limit=3", "owner", false, nil, 200, 1},
		{"bad limit", "GET", "/notifications?limit=101", "owner", false, nil, 400, 0},
		{"bad cursor", "GET", "/notifications?cursor=invalid", "owner", false, nil, 400, 0},
		{"count", "GET", "/notifications/unread-count", "owner", false, nil, 200, 1},
		{"read", "PUT", "/notifications/" + id + "/read", "owner", true, nil, 200, 1},
		{"foreign or missing", "PUT", "/notifications/" + id + "/read", "owner", false, nil, 404, 1},
		{"invalid id", "PUT", "/notifications/bad/read", "owner", false, nil, 400, 0},
		{"failure", "GET", "/notifications", "owner", false, errors.New("secret"), 500, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			fake := &inboxFake{found: tt.found, err: tt.err}
			middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error { c.Set("user_id", tt.user); return next(c) }
			}
			NewInboxRouter(fake).RegisterRoutes(e.Group("/api/v1"), middleware)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(tt.method, "/api/v1"+tt.path, nil))
			if rec.Code != tt.status || fake.calls != tt.calls {
				t.Fatalf("status=%d calls=%d body=%s", rec.Code, fake.calls, rec.Body.String())
			}
			if fake.calls > 0 && fake.user != tt.user {
				t.Fatalf("user=%q", fake.user)
			}
			if tt.name == "list" && fake.limit != 3 {
				t.Fatalf("limit=%d", fake.limit)
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("internal error leaked")
			}
		})
	}
}

type ownedInbox struct {
	owner, id string
	read      bool
	changes   int
}

func (f *ownedInbox) List(context.Context, string, int, *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (f *ownedInbox) UnreadCount(context.Context, string) (int64, error) { return 0, nil }
func (f *ownedInbox) MarkRead(_ context.Context, user, id string) (bool, error) {
	if user != f.owner || id != f.id {
		return false, nil
	}
	if !f.read {
		f.read = true
		f.changes++
	}
	return true, nil
}
func (f *ownedInbox) Create(_ context.Context, _ *entities.Notification) error {
	return nil
}
func (f *ownedInbox) CreateBatch(_ context.Context, _ []entities.Notification) error {
	return nil
}

func TestInboxReadOwnershipAndRepeat(t *testing.T) {
	id := "b2a3a528-4591-4bee-bfbb-88e06965d87c"
	repo := &ownedInbox{owner: "owner", id: id}
	e := echo.New()
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error { c.Set("user_id", c.Request().Header.Get("X-Test-User")); return next(c) }
	}
	NewInboxRouter(repo).RegisterRoutes(e.Group("/api/v1"), auth)
	for _, step := range []struct {
		user    string
		status  int
		changes int
	}{{"foreign", 404, 0}, {"owner", 200, 1}, {"owner", 200, 1}, {"foreign", 404, 1}} {
		req := httptest.NewRequest("PUT", "/api/v1/notifications/"+id+"/read", nil)
		req.Header.Set("X-Test-User", step.user)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != step.status || repo.changes != step.changes {
			t.Fatalf("user=%s status=%d changes=%d", step.user, rec.Code, repo.changes)
		}
	}
}
