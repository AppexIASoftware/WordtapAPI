package queries_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestListTeacherCourseReviewsEnforcesOwnership(t *testing.T) {
	h := queries.NewListTeacherCourseReviewsHandler(&reviewRepoStub{})
	if _, err := h.Handle(context.Background(), queries.ListTeacherCourseReviewsQuery{InstructorID: "teacher-b", RequesterID: "teacher-a", RequesterRole: entities.RoleInstructor}); err == nil {
		t.Fatal("expected cross-teacher history access to be denied")
	}
	if _, err := h.Handle(context.Background(), queries.ListTeacherCourseReviewsQuery{InstructorID: "teacher-a", RequesterID: "teacher-a", RequesterRole: entities.RoleInstructor}); err != nil {
		t.Fatalf("owner history failed: %v", err)
	}
}
