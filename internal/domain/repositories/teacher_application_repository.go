package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type TeacherApplicationRepository interface {
	Create(ctx context.Context, app *entities.TeacherApplication) error
	FindByID(ctx context.Context, id string) (*entities.TeacherApplication, error)
	FindByUserID(ctx context.Context, userID string) (*entities.TeacherApplication, error)
	List(ctx context.Context, status *entities.ApplicationStatus, limit, offset int) ([]entities.TeacherApplication, int64, error)
	Update(ctx context.Context, app *entities.TeacherApplication) error
	ApproveInTx(ctx context.Context, appID string, reviewerID string) error
	RejectInTx(ctx context.Context, appID string, reviewerID string, reason string) error
	SuspendInTx(ctx context.Context, appID string, reviewerID string, reason string) error
	ReactivateInTx(ctx context.Context, appID string, reviewerID string) error
	DeleteInTx(ctx context.Context, appID string) error
}
