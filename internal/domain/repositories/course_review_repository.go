package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// CourseReviewRepository owns atomic course-review lifecycle transitions.
type CourseReviewRepository interface {
	CreateSubmission(context.Context, *entities.CourseReviewRequest) error
	ListPending(context.Context) ([]entities.CourseReviewRequest, error)
	Decide(context.Context, string, string, entities.CourseReviewStatus, *string) (*entities.CourseReviewRequest, error)
	Withdraw(context.Context, string, string) (*entities.CourseReviewRequest, error)
	ListByInstructor(context.Context, string) ([]entities.CourseReviewRequest, error)
}
