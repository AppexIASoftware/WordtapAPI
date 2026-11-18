package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type MySQLTeacherApplicationRepository struct {
	db *gorm.DB
}

func NewMySQLTeacherApplicationRepository(db *gorm.DB) domainRepo.TeacherApplicationRepository {
	return &MySQLTeacherApplicationRepository{db: db}
}

func (r *MySQLTeacherApplicationRepository) Create(ctx context.Context, app *entities.TeacherApplication) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *MySQLTeacherApplicationRepository) FindByID(ctx context.Context, id string) (*entities.TeacherApplication, error) {
	var app entities.TeacherApplication
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Reviewer").
		Where("id = ?", id).
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *MySQLTeacherApplicationRepository) FindByUserID(ctx context.Context, userID string) (*entities.TeacherApplication, error) {
	var app entities.TeacherApplication
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Reviewer").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *MySQLTeacherApplicationRepository) List(ctx context.Context, status *entities.ApplicationStatus, limit, offset int) ([]entities.TeacherApplication, int64, error) {
	var apps []entities.TeacherApplication
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.TeacherApplication{})
	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	err := query.
		Preload("User").
		Preload("Reviewer").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&apps).Error
	if err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}

func (r *MySQLTeacherApplicationRepository) Update(ctx context.Context, app *entities.TeacherApplication) error {
	return r.db.WithContext(ctx).Save(app).Error
}

func lockTeacherAppAndUser(tx *gorm.DB, appID string) (*entities.TeacherApplication, *entities.User, error) {
	var app entities.TeacherApplication
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", appID).First(&app).Error; err != nil {
		return nil, nil, err
	}
	var user entities.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", app.UserID).First(&user).Error; err != nil {
		return nil, nil, err
	}
	if user.Role == entities.RoleAdmin {
		return nil, nil, errors.New("cannot modify an administrator account through teacher application workflow")
	}
	return &app, &user, nil
}

func reviewerPtr(reviewerID string) *string {
	id := strings.TrimSpace(reviewerID)
	if id == "" {
		return nil
	}
	return &id
}

func (r *MySQLTeacherApplicationRepository) ApproveInTx(ctx context.Context, appID string, reviewerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		app, user, err := lockTeacherAppAndUser(tx, appID)
		if err != nil {
			return err
		}
		if app.Status == entities.ApplicationStatusApproved {
			return errors.New("application is already approved")
		}

		now := time.Now().UTC()
		app.Status = entities.ApplicationStatusApproved
		app.ReviewedBy = reviewerPtr(reviewerID)
		app.ReviewedAt = &now
		app.UpdatedAt = now
		if err := tx.Save(app).Error; err != nil {
			return err
		}

		// Protect moderator accounts from unintended role downgrade while restoring is_active if needed
		if user.Role != entities.RoleModerator || !user.IsActive {
			updates := map[string]any{
				"is_active":  true,
				"updated_at": now,
			}
			if user.Role != entities.RoleModerator {
				updates["role"] = entities.RoleInstructor
			}
			if err := tx.Model(&entities.User{}).Where("id = ?", app.UserID).Updates(updates).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *MySQLTeacherApplicationRepository) RejectInTx(ctx context.Context, appID string, reviewerID string, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		app, _, err := lockTeacherAppAndUser(tx, appID)
		if err != nil {
			return err
		}
		if app.Status == entities.ApplicationStatusApproved || app.Status == entities.ApplicationStatusSuspended {
			return errors.New("cannot reject an already approved or suspended application")
		}

		now := time.Now().UTC()
		var reasonPtr *string
		if reason != "" {
			reasonPtr = &reason
		}
		app.Status = entities.ApplicationStatusRejected
		app.ReviewedBy = reviewerPtr(reviewerID)
		app.ReviewedAt = &now
		app.RejectionReason = reasonPtr
		app.UpdatedAt = now
		return tx.Save(app).Error
	})
}

func (r *MySQLTeacherApplicationRepository) SuspendInTx(ctx context.Context, appID string, reviewerID string, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		app, user, err := lockTeacherAppAndUser(tx, appID)
		if err != nil {
			return err
		}
		if app.Status != entities.ApplicationStatusApproved {
			return errors.New("only approved applications can be suspended")
		}
		if user.Role == entities.RoleModerator {
			return errors.New("cannot suspend a moderator account through teacher application workflow")
		}

		now := time.Now().UTC()
		var reasonPtr *string
		if reason != "" {
			reasonPtr = &reason
		}

		app.Status = entities.ApplicationStatusSuspended
		app.ReviewedBy = reviewerPtr(reviewerID)
		app.ReviewedAt = &now
		app.RejectionReason = reasonPtr
		app.UpdatedAt = now
		if err := tx.Save(app).Error; err != nil {
			return err
		}

		// Suspender la cuenta de usuario
		if err := tx.Model(&entities.User{}).Where("id = ?", app.UserID).Updates(map[string]any{
			"is_active":  false,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		// Invalidar todas las sesiones JWT activas
		if err := tx.Model(&entities.UserSession{}).Where("user_id = ? AND revoked_at IS NULL", app.UserID).Update("revoked_at", &now).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *MySQLTeacherApplicationRepository) ReactivateInTx(ctx context.Context, appID string, reviewerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		app, user, err := lockTeacherAppAndUser(tx, appID)
		if err != nil {
			return err
		}
		if app.Status != entities.ApplicationStatusSuspended && app.Status != entities.ApplicationStatusRejected {
			return errors.New("only suspended or rejected applications can be reactivated")
		}

		now := time.Now().UTC()
		app.Status = entities.ApplicationStatusApproved
		app.ReviewedBy = reviewerPtr(reviewerID)
		app.ReviewedAt = &now
		app.UpdatedAt = now
		if err := tx.Save(app).Error; err != nil {
			return err
		}

		// Reactivar la cuenta preservando rol si ya es moderador
		updates := map[string]any{
			"is_active":  true,
			"updated_at": now,
		}
		if user.Role != entities.RoleModerator {
			updates["role"] = entities.RoleInstructor
		}
		if err := tx.Model(&entities.User{}).Where("id = ?", app.UserID).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *MySQLTeacherApplicationRepository) DeleteInTx(ctx context.Context, appID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		app, user, err := lockTeacherAppAndUser(tx, appID)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		// Revertir rol solo si se elimina la única postulación aprobada activa del instructor
		if app.Status == entities.ApplicationStatusApproved && user.Role == entities.RoleInstructor {
			var otherApproved int64
			if err := tx.Model(&entities.TeacherApplication{}).
				Where("user_id = ? AND id <> ? AND status = ?", app.UserID, app.ID, entities.ApplicationStatusApproved).
				Count(&otherApproved).Error; err != nil {
				return err
			}
			if otherApproved == 0 {
				if err := tx.Model(&entities.User{}).Where("id = ? AND role = ?", app.UserID, entities.RoleInstructor).Updates(map[string]any{
					"role":       entities.RoleStudent,
					"updated_at": now,
				}).Error; err != nil {
					return err
				}
			}
		}

		// Eliminar la postulación
		return tx.Delete(app).Error
	})
}
