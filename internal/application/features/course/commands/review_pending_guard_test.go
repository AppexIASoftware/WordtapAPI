package commands_test

import (
    "context"
    "testing"

    "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
    "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestUpdateCourseRejectsContentWhileReviewPending(t *testing.T) {
    owner := "teacher"
    title := "Changed"
    course := &entities.Course{ID: "course", Title: "Original", CreatedBy: &owner, Status: entities.ContentStatusInReview}
    repo := &submissionCourseRepo{course: course}
    _, err := commands.NewUpdateCourseHandler(repo).Handle(context.Background(), commands.UpdateCourseCommand{
        CourseID: course.ID, Title: &title, RequesterID: owner, RequesterRole: entities.RoleInstructor,
    })
    if err == nil {
        t.Fatal("content changes must be rejected while a course is in review")
    }
    if course.Title != "Original" {
        t.Fatalf("pending course was mutated: %q", course.Title)
    }
}
