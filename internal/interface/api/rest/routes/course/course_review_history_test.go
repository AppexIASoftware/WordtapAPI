package course_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestReviewHistoryRoute(t *testing.T) {
	e, jwt, _, repo := setupTestServer()
	repo.requests = []entities.CourseReviewRequest{
		{ID: "pending", Status: entities.CourseReviewStatusPending},
		{ID: "approved", Status: entities.CourseReviewStatusApproved},
		{ID: "rejected", Status: entities.CourseReviewStatusRejected},
		{ID: "withdrawn", Status: entities.CourseReviewStatusWithdrawn},
	}
	for _, tc := range []struct {
		name   string
		role   entities.UserRole
		status int
	}{
		{"administrator", entities.RoleAdmin, http.StatusOK},
		{"moderator", entities.RoleModerator, http.StatusOK},
		{"teacher", entities.RoleInstructor, http.StatusForbidden},
		{"student", entities.RoleStudent, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, _, err := jwt.GenerateAccessToken(&entities.User{ID: tc.name, Email: tc.name + "@example.com", Role: tc.role, AccessTier: entities.AccessTierFree})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/course-reviews/history", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if tc.status == http.StatusOK {
				var items []entities.CourseReviewRequest
				if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil || len(items) != 3 {
					t.Fatalf("history=%s err=%v", rec.Body.String(), err)
				}
			}
		})
	}
}
