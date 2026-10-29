package course_test

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestCourseCategoryUpdate(t *testing.T) {
 for _, tc := range []struct{name, body, want string}{
  {"omitted", `{}`, "category-1"},
  {"null", `{"category_id":null}`, ""},
  {"value", `{"category_id":"category-2"}`, "category-2"},
 } {
  t.Run(tc.name, func(t *testing.T) {
   e, jwt, repo, _ := setupTestServer()
   owner, category := "teacher-123", "category-1"
   repo.courses = append(repo.courses, entities.Course{ID:"course-1", CreatedBy:&owner, OwnerType:entities.OwnerTypeTeacher, CategoryID:&category})
   token, _, _ := jwt.GenerateAccessToken(&entities.User{ID:owner, Email:"teacher@wordtap.app", Role:entities.RoleInstructor})
   req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/course-1", strings.NewReader(tc.body))
   req.Header.Set("Authorization", "Bearer "+token)
   req.Header.Set("Content-Type", "application/json")
   rec := httptest.NewRecorder()
   e.ServeHTTP(rec, req)
   if rec.Code != http.StatusOK { t.Fatalf("status %d: %s", rec.Code, rec.Body.String()) }
   got := repo.courses[0].CategoryID
   if tc.want == "" { if got != nil { t.Fatalf("expected nil, got %q", *got) }; return }
   if got == nil || *got != tc.want { t.Fatalf("expected %s, got %v", tc.want, got) }
  })
 }
}
