package commands_test

import (
 "context"
 "errors"
 "testing"

 "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
 "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type rejectUpdateRepo struct { *submissionCourseRepo; updates int }
func (r *rejectUpdateRepo) Update(context.Context, *entities.Course) error { r.updates++; return errors.New("unexpected stale save") }

func TestSubmissionDoesNotSaveStaleCourseAfterCommit(t *testing.T) {
 owner := "teacher"
 course := &entities.Course{ID:"course", CreatedBy:&owner, Status:entities.ContentStatusDraft}
 repo := &rejectUpdateRepo{submissionCourseRepo:&submissionCourseRepo{course:course}}
 review := &reviewCourseRepo{course:course}
 result, err := commands.NewSubmitCourseReviewHandler(repo, review).Handle(context.Background(), commands.SubmitCourseReviewCommand{CourseID:course.ID, RequesterID:owner, RequesterRole:entities.RoleInstructor})
 if err != nil || repo.updates != 0 || result == nil || result.Status != entities.ContentStatusInReview { t.Fatalf("submission must not save stale course: result=%+v updates=%d err=%v", result, repo.updates, err) }
}
