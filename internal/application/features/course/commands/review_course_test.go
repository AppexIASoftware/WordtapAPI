package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestReviewCourseRejectsNonAdminAndInvalidDecision(t *testing.T) {
	h := commands.NewReviewCourseHandler(&reviewCourseRepo{})
	for _, cmd := range []commands.ReviewCourseCommand{
		{ReviewID: "review-1", ReviewerID: "teacher", ReviewerRole: entities.RoleInstructor, Status: entities.CourseReviewStatusApproved},
		{ReviewID: "review-1", ReviewerID: "admin", ReviewerRole: entities.RoleAdmin, Status: entities.CourseReviewStatusPending},
	} {
		if _, err := h.Handle(context.Background(), cmd); err == nil {
			t.Fatalf("expected command rejection for %+v", cmd)
		}
	}
}
