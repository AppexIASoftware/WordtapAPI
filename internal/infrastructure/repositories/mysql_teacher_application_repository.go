package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

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

func (r *MySQLTeacherApplicationRepository) ApproveInTx(ctx context.Context, appID string, reviewerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app entities.TeacherApplication
		if err := tx.Where("id = ?", appID).First(&app).Error; err != nil {
			return err
		}
		if app.Status == entities.ApplicationStatusApproved {
			return errors.New("application is already approved")
		}

		now := time.Now()
		app.Status = entities.ApplicationStatusApproved
		app.ReviewedBy = &reviewerID
		app.ReviewedAt = &now
		app.UpdatedAt = now
		if err := tx.Save(&app).Error; err != nil {
			return err
		}

		// Promover al usuario a rol de instructor
		if err := tx.Model(&entities.User{}).Where("id = ?", app.UserID).Updates(map[string]any{
			"role":       entities.RoleInstructor,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *MySQLTeacherApplicationRepository) RejectInTx(ctx context.Context, appID string, reviewerID string, reason string) error {
	now := time.Now()
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	return r.db.WithContext(ctx).Model(&entities.TeacherApplication{}).Where("id = ?", appID).Updates(map[string]any{
		"status":           entities.ApplicationStatusRejected,
		"reviewed_by":      reviewerID,
		"reviewed_at":      now,
		"rejection_reason": reasonPtr,
		"updated_at":       now,
	}).Error
}

func (r *MySQLTeacherApplicationRepository) SuspendInTx(ctx context.Context, appID string, reviewerID string, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app entities.TeacherApplication
		if err := tx.Where("id = ?", appID).First(&app).Error; err != nil {
			return err
		}

		now := time.Now()
		var reasonPtr *string
		if reason != "" {
			reasonPtr = &reason
		}

		app.Status = entities.ApplicationStatusSuspended
		app.ReviewedBy = &reviewerID
		app.ReviewedAt = &now
		app.RejectionReason = reasonPtr
		app.UpdatedAt = now
		if err := tx.Save(&app).Error; err != nil {
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
		var app entities.TeacherApplication
		if err := tx.Where("id = ?", appID).First(&app).Error; err != nil {
			return err
		}

		now := time.Now()
		app.Status = entities.ApplicationStatusApproved
		app.ReviewedBy = &reviewerID
		app.ReviewedAt = &now
		app.UpdatedAt = now
		if err := tx.Save(&app).Error; err != nil {
			return err
		}

		// Reactivar la cuenta y asegurar rol de instructor
		if err := tx.Model(&entities.User{}).Where("id = ?", app.UserID).Updates(map[string]any{
			"is_active":  true,
			"role":       entities.RoleInstructor,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *MySQLTeacherApplicationRepository) DeleteInTx(ctx context.Context, appID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app entities.TeacherApplication
		if err := tx.Where("id = ?", appID).First(&app).Error; err != nil {
			return err
		}

		now := time.Now()
		// Revertir rol a estudiante si era instructor
		_ = tx.Model(&entities.User{}).Where("id = ? AND role = ?", app.UserID, entities.RoleInstructor).Updates(map[string]any{
			"role":       entities.RoleStudent,
			"updated_at": now,
		}).Error

		// Eliminar la postulación
		return tx.Delete(&app).Error
	})
}
