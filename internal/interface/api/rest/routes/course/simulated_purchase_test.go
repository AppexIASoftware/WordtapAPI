package course

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/labstack/echo/v5"
)

type purchaseRouteStub struct {
	user, course, key string
	err               error
}

func (s *purchaseRouteStub) Purchase(_ context.Context, user, course, key string) (*commands.SimulatedCoursePurchase, error) {
	s.user, s.course, s.key = user, course, key
	if s.err != nil {
		return nil, s.err
	}
	return &commands.SimulatedCoursePurchase{ID: "receipt", UserID: user, CourseID: course, AmountCents: 2700, Currency: "PEN"}, nil
}
func (s *purchaseRouteStub) Find(_ context.Context, user, course string) (*commands.SimulatedCoursePurchase, error) {
	s.user, s.course = user, course
	return nil, s.err
}

func TestSimulatedPurchaseHTTP(t *testing.T) {
	jwt := security.NewJWTService("test-secret-for-simulated-purchase", time.Hour, time.Hour)
	student, _, _ := jwt.GenerateAccessToken(&entities.User{ID: "student-1", Role: entities.RoleStudent})
	teacher, _, _ := jwt.GenerateAccessToken(&entities.User{ID: "teacher-1", Role: entities.RoleInstructor})
	for _, tc := range []struct {
		name, token, key, body string
		enabled                bool
		repoErr                error
		status                 int
	}{
		{"anonymous", "", "key", "", true, nil, 401},
		{"teacher forbidden", teacher, "key", "", true, nil, 403},
		{"disabled", student, "key", "", false, nil, 404},
		{"key required", student, "", "", true, nil, 400},
		{"key conflict", student, "key", "", true, commands.ErrIdempotencyConflict, 409},
		{"ineligible", student, "key", "", true, commands.ErrCourseNotPurchasable, 404},
		{"server price and identity", student, "key", `{"user_id":"attacker","amount_cents":1,"currency":"USD"}`, true, nil, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &purchaseRouteStub{err: tc.repoErr}
			e := echo.New()
			NewSimulatedPurchaseRouter(commands.NewSimulatePurchaseHandler(repo, tc.enabled)).RegisterRoutes(e.Group("/api/v1"), middleware.RequireAuth(jwt), middleware.RequireRole(entities.RoleStudent))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/student/courses/course-1/simulated-purchase", strings.NewReader(tc.body))
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			req.Header.Set("Idempotency-Key", tc.key)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if tc.status == 200 && (repo.user != "student-1" || repo.course != "course-1" || !strings.Contains(rec.Body.String(), `"amount_cents":2700`) || !strings.Contains(rec.Body.String(), `"currency":"PEN"`)) {
				t.Fatalf("unexpected purchase: %+v %s", repo, rec.Body.String())
			}
		})
	}
}
