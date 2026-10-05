package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type reviewCourseRepo struct {
	course  *entities.Course
	request *entities.CourseReviewRequest
}

func (r *reviewCourseRepo) CreateSubmission(_ context.Context, req *entities.CourseReviewRequest) error {
	r.request = req
	r.course.Status = entities.ContentStatusInReview
	return nil
}
func (r *reviewCourseRepo) ListPending(context.Context) ([]entities.CourseReviewRequest, error) {
	return nil, nil
}
func (r *reviewCourseRepo) Decide(_ context.Context, id, admin string, status entities.CourseReviewStatus, notes *string) (*entities.CourseReviewRequest, error) {
	if r.request == nil || r.request.ID != id || r.request.Status != entities.CourseReviewStatusPending {
		return nil, context.Canceled
	}
	now := time.Now()
	r.request.Status = status
	r.request.ReviewerAdminID = &admin
	r.request.FeedbackNotes = notes
	r.request.ReviewedAt = &now
	return r.request, nil
}
func (r *reviewCourseRepo) ListByInstructor(context.Context, string) ([]entities.CourseReviewRequest, error) {
	return nil, nil
}

type submissionCourseRepo struct{ course *entities.Course }

func (r *submissionCourseRepo) FindByID(context.Context, string) (*entities.Course, error) {
	return r.course, nil
}
func (r *submissionCourseRepo) Create(context.Context, *entities.Course) error { return nil }
func (r *submissionCourseRepo) FindByAuthor(context.Context, string) ([]entities.Course, error) {
	return nil, nil
}
func (r *submissionCourseRepo) FindPublished(context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (r *submissionCourseRepo) Update(context.Context, *entities.Course) error { return nil }
func (r *submissionCourseRepo) Delete(context.Context, string) error           { return nil }

func TestSubmitCourseReviewPersistsPendingRequest(t *testing.T) {
	author := "teacher-1"
	course := &entities.Course{ID: "course-1", CreatedBy: &author, Status: entities.ContentStatusDraft}
	reviews := &reviewCourseRepo{course: course}
	handler := commands.NewSubmitCourseReviewHandler(&submissionCourseRepo{course}, reviews)
	result, err := handler.Handle(context.Background(), commands.SubmitCourseReviewCommand{CourseID: course.ID, RequesterID: author, RequesterRole: entities.RoleInstructor})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != entities.ContentStatusInReview || reviews.request == nil || reviews.request.Status != entities.CourseReviewStatusPending {
		t.Fatalf("submission was not persisted: course=%+v request=%+v", result, reviews.request)
	}
}

func TestReviewCourseRequiresPendingRequestAndPersistsDecision(t *testing.T) {
	admin := "admin-1"
	req := &entities.CourseReviewRequest{ID: "review-1", CourseID: "course-1", InstructorID: "teacher-1", Status: entities.CourseReviewStatusPending}
	repo := &reviewCourseRepo{request: req}
	handler := commands.NewReviewCourseHandler(repo)
	result, err := handler.Handle(context.Background(), commands.ReviewCourseCommand{ReviewID: req.ID, ReviewerID: admin, ReviewerRole: entities.RoleAdmin, Status: entities.CourseReviewStatusApproved})
	if err != nil {
		t.Fatal(err)
	}
	if result.ReviewerAdminID == nil || *result.ReviewerAdminID != admin || result.Status != entities.CourseReviewStatusApproved || result.ReviewedAt == nil {
		t.Fatalf("review metadata missing: %+v", result)
	}
}
