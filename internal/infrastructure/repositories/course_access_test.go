package repositories

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestCanGrantTestCourseAccess(t *testing.T) {
	student := entities.User{Role: entities.RoleStudent, IsActive: true}
	course := entities.Course{OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished, AccessTier: entities.AccessTierCourse, PriceCents: 100}
	if !canGrantTestCourseAccess(student, course) {
		t.Fatal("eligible student and course were rejected")
	}
	student.IsActive = false
	if canGrantTestCourseAccess(student, course) {
		t.Fatal("inactive student was accepted")
	}
	student.IsActive = true
	student.Role = entities.RoleInstructor
	if canGrantTestCourseAccess(student, course) {
		t.Fatal("non-student was accepted")
	}
	student.Role = entities.RoleStudent
	course.Status = entities.ContentStatusDraft
	if canGrantTestCourseAccess(student, course) {
		t.Fatal("draft course was accepted")
	}
	course.Status = entities.ContentStatusPublished
	course.OwnerType = entities.OwnerTypePlatform
	if canGrantTestCourseAccess(student, course) {
		t.Fatal("platform course was accepted")
	}
}

func TestCourseAccessSourceAllowed(t *testing.T) {
	for _, tc := range []struct {
		status, source  string
		allowTest, want bool
	}{
		{"active", "test", true, true},
		{"active", "test", false, false},
		{"revoked", "test", true, false},
		{"active", "unknown", true, false},
	} {
		if got := courseAccessAllowed(courseAccess{Status: tc.status, Source: tc.source}, tc.allowTest); got != tc.want {
			t.Fatalf("status=%s source=%s allowTest=%v: got %v", tc.status, tc.source, tc.allowTest, got)
		}
	}
}
