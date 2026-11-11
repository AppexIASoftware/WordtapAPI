package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	app "github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationprefs"
	"github.com/labstack/echo/v5"
)

type testRepo struct {
	user        string
	input       app.Update
	err         error
	calls       int
	preferences *app.Preferences
}

func (r *testRepo) Get(_ context.Context, id string) (*app.Preferences, error) {
	r.user = id
	r.calls++
	if r.preferences != nil {
		return r.preferences, r.err
	}
	return &app.Preferences{}, r.err
}
func (r *testRepo) Put(_ context.Context, id string, input app.Update) (*app.Preferences, error) {
	r.user = id
	r.input = input
	r.calls++
	return &app.Preferences{}, r.err
}

func TestPreferencesRoutes(t *testing.T) {
	valid := `{"enabled":false,"daily_word_enabled":false,"reminder_enabled":false,"timezone":"UTC","mode":"random"}`
	for _, tt := range []struct {
		name, method, body, user string
		repoErr                  error
		status, calls            int
	}{
		{"missing identity", "GET", "", "", nil, 401, 0},
		{"get own preferences", "GET", "", "user-a", nil, 200, 1},
		{"put false values", "PUT", valid, "user-b", nil, 200, 1},
		{"reject user id", "PUT", `{"user_id":"victim"}`, "user-a", nil, 400, 0},
		{"reject invalid data", "PUT", `{"mode":"random"}`, "user-a", nil, 400, 0},
		{"reject omitted schedule enabled", "PUT", `{"enabled":true,"daily_word_enabled":true,"reminder_enabled":true,"timezone":"UTC","mode":"custom","schedules":[{"time":"09:00","days":[1]}]}`, "user-a", nil, 400, 0},
		{"reject null enabled custom", "PUT", `{"enabled":true,"daily_word_enabled":true,"reminder_enabled":true,"timezone":"UTC","mode":"custom","schedules":[{"time":"09:00","days":[1],"enabled":null}]}`, "user-a", nil, 400, 0},
		{"reject omitted enabled random", "PUT", `{"enabled":true,"daily_word_enabled":true,"reminder_enabled":true,"timezone":"UTC","mode":"random","schedules":[{"time":"09:00","days":[1]}]}`, "user-a", nil, 400, 0},
		{"reject null schedule enabled", "PUT", `{"enabled":true,"daily_word_enabled":true,"reminder_enabled":true,"timezone":"UTC","mode":"random","schedules":[{"time":"09:00","days":[1],"enabled":null}]}`, "user-a", nil, 400, 0},
		{"reject changed random policy", "PUT", `{"enabled":true,"daily_word_enabled":true,"reminder_enabled":true,"timezone":"UTC","mode":"random","random_count":4}`, "user-a", nil, 400, 0},
		{"reject trailing json", "PUT", valid + `{}`, "user-a", nil, 400, 0},
		{"reject oversized", "PUT", valid + strings.Repeat(" ", 16385), "user-a", nil, 400, 0},
		{"database error", "GET", "", "user-a", errors.New("secret"), 500, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			repo := &testRepo{err: tt.repoErr}
			auth := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error { c.Set("user_id", tt.user); return next(c) }
			}
			NewRouter(repo).RegisterRoutes(e.Group("/api/v1"), auth)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/api/v1/notification-preferences", strings.NewReader(tt.body))
			e.ServeHTTP(rec, req)
			if rec.Code != tt.status || repo.calls != tt.calls {
				t.Fatalf("status=%d calls=%d body=%s", rec.Code, repo.calls, rec.Body.String())
			}
			if repo.calls > 0 && repo.user != tt.user {
				t.Fatalf("user = %q", repo.user)
			}
			if tt.name == "put false values" && (repo.input.Enabled == nil || *repo.input.Enabled) {
				t.Fatal("false value lost")
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("internal error leaked")
			}
		})
	}
}

func TestGetDataRoundTripsToPut(t *testing.T) {
	for _, mode := range []string{app.ModeRandom, app.ModeCustom} {
		t.Run(mode, func(t *testing.T) {
			original := &app.Preferences{Enabled: false, DailyWordEnabled: true, ReminderEnabled: true, Timezone: "UTC", Mode: mode, Schedules: []app.Schedule{{Time: "09:00", Days: []int{1, 2}, Enabled: false}}, RandomCount: 3, RandomStart: "07:00", RandomEnd: "22:00"}
			repo := &testRepo{preferences: original}
			e := echo.New()
			auth := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error { c.Set("user_id", "owner"); return next(c) }
			}
			NewRouter(repo).RegisterRoutes(e.Group("/api/v1"), auth)
			get := httptest.NewRecorder()
			e.ServeHTTP(get, httptest.NewRequest("GET", "/api/v1/notification-preferences", nil))
			var response struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(get.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			put := httptest.NewRecorder()
			e.ServeHTTP(put, httptest.NewRequest("PUT", "/api/v1/notification-preferences", strings.NewReader(string(response.Data))))
			if put.Code != 200 {
				t.Fatalf("GET data rejected by PUT: %d %s", put.Code, put.Body.String())
			}
			if len(repo.input.Schedules) != 1 || repo.input.Schedules[0].Enabled == nil || *repo.input.Schedules[0].Enabled {
				t.Fatal("retained disabled schedule was not decoded")
			}
		})
	}
}
