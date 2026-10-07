package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestWithdrawCourseReviewRejectsNonInstructorOrEmptyID(t *testing.T) {
	h := commands.NewWithdrawCourseReviewHandler(&reviewCourseRepo{})
	for _, cmd := range []commands.WithdrawCourseReviewCommand{
		{ID: "", RequesterID: "teacher", RequesterRole: entities.RoleInstructor},
		{ID: "course-1", RequesterID: "teacher", RequesterRole: entities.RoleAdmin},
		{ID: "course-1", RequesterID: "", RequesterRole: entities.RoleInstructor},
	} {
		if _, err := h.Handle(context.Background(), cmd); err == nil {
			t.Fatalf("expected command rejection for %+v", cmd)
		}
	}
}

func TestWithdrawCourseReviewSuccess(t *testing.T) {
	course := &entities.Course{ID: "course-1", Status: entities.ContentStatusInReview}
	req := &entities.CourseReviewRequest{ID: "rev-1", CourseID: "course-1", InstructorID: "teacher-1", Status: entities.CourseReviewStatusPending}
	repo := &reviewCourseRepo{course: course, request: req}

	h := commands.NewWithdrawCourseReviewHandler(repo)
	res, err := h.Handle(context.Background(), commands.WithdrawCourseReviewCommand{
		ID:            "course-1",
		RequesterID:   "teacher-1",
		RequesterRole: entities.RoleInstructor,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != entities.CourseReviewStatusWithdrawn {
		t.Fatalf("expected status withdrawn, got %v", res.Status)
	}
	if course.Status != entities.ContentStatusDraft {
		t.Fatalf("expected course status draft, got %v", course.Status)
	}
}
