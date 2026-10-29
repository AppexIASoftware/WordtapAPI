package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// PublishedCourseEditRepository atomically saves an edited published course and its review request.
type PublishedCourseEditRepository interface {
	UpdatePublishedAndSubmit(context.Context, *entities.Course, *entities.CourseReviewRequest) error
}

// CourseReviewRepository owns atomic course-review lifecycle transitions.
type CourseReviewRepository interface {
	CreateSubmission(context.Context, *entities.CourseReviewRequest) error
	ListPending(context.Context) ([]entities.CourseReviewRequest, error)
	Decide(context.Context, string, string, entities.CourseReviewStatus, *string) (*entities.CourseReviewRequest, error)
	Withdraw(context.Context, string, string) (*entities.CourseReviewRequest, error)
	ListByInstructor(context.Context, string) ([]entities.CourseReviewRequest, error)
}
