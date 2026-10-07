package queries_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type reviewRepoStub struct {
	requests []entities.CourseReviewRequest
}

func (s *reviewRepoStub) CreateSubmission(context.Context, *entities.CourseReviewRequest) error {
	return nil
}
func (s *reviewRepoStub) ListPending(context.Context) ([]entities.CourseReviewRequest, error) {
	return s.requests, nil
}
func (s *reviewRepoStub) Decide(context.Context, string, string, entities.CourseReviewStatus, *string) (*entities.CourseReviewRequest, error) {
	return nil, nil
}
func (s *reviewRepoStub) Withdraw(context.Context, string, string) (*entities.CourseReviewRequest, error) {
	return nil, nil
}
func (s *reviewRepoStub) ListByInstructor(context.Context, string) ([]entities.CourseReviewRequest, error) {
	return s.requests, nil
}

var _ repositories.CourseReviewRepository = (*reviewRepoStub)(nil)

func TestListCourseReviewsRequiresAdmin(t *testing.T) {
	h := queries.NewListCourseReviewsHandler(&reviewRepoStub{})
	if _, err := h.Handle(context.Background(), queries.ListCourseReviewsQuery{RequesterID: "teacher", RequesterRole: entities.RoleInstructor}); err == nil {
		t.Fatal("expected non-admin to be denied")
	}
	if _, err := h.Handle(context.Background(), queries.ListCourseReviewsQuery{RequesterID: "admin", RequesterRole: entities.RoleAdmin}); err != nil {
		t.Fatalf("admin list failed: %v", err)
	}
}
