package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type MySQLCourseReviewRepository struct{ db *gorm.DB }

func NewMySQLCourseReviewRepository(db *gorm.DB) domainRepo.CourseReviewRepository {
	return &MySQLCourseReviewRepository{db: db}
}

func (r *MySQLCourseReviewRepository) CreateSubmission(ctx context.Context, request *entities.CourseReviewRequest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entities.Course{}).Where("id = ? AND created_by = ? AND status = ?", request.CourseID, request.InstructorID, entities.ContentStatusDraft).Updates(map[string]any{"status": entities.ContentStatusInReview, "updated_at": request.SubmittedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("course is unavailable for submission")
		}
		return tx.Create(request).Error
	})
}

func (r *MySQLCourseReviewRepository) ListPending(ctx context.Context) ([]entities.CourseReviewRequest, error) {
	items := make([]entities.CourseReviewRequest, 0)
	err := r.db.WithContext(ctx).Preload("Course").Preload("Instructor").Where("status = ?", entities.CourseReviewStatusPending).Order("submitted_at ASC").Find(&items).Error
	return items, err
}

func (r *MySQLCourseReviewRepository) Decide(ctx context.Context, id, adminID string, status entities.CourseReviewStatus, notes *string) (*entities.CourseReviewRequest, error) {
	var request entities.CourseReviewRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", id, entities.CourseReviewStatusPending).First(&request).Error; err != nil {
			return errors.New("review request is not pending")
		}
		now := time.Now()
		request.Status = status
		request.ReviewerAdminID = &adminID
		request.FeedbackNotes = notes
		request.ReviewedAt = &now
		result := tx.Model(&entities.CourseReviewRequest{}).Where("id = ? AND status = ?", id, entities.CourseReviewStatusPending).Updates(map[string]any{"status": status, "reviewer_admin_id": adminID, "feedback_notes": notes, "reviewed_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("review request is no longer pending")
		}
		courseStatus := entities.ContentStatusDraft
		if status == entities.CourseReviewStatusApproved {
			courseStatus = entities.ContentStatusPublished
		}
		course := tx.Model(&entities.Course{}).Where("id = ? AND status = ?", request.CourseID, entities.ContentStatusInReview).Updates(map[string]any{"status": courseStatus, "updated_at": now})
		if course.Error != nil {
			return course.Error
		}
		if course.RowsAffected != 1 {
			return errors.New("course is not in review")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *MySQLCourseReviewRepository) ListByInstructor(ctx context.Context, instructorID string) ([]entities.CourseReviewRequest, error) {
	items := make([]entities.CourseReviewRequest, 0)
	err := r.db.WithContext(ctx).Preload("Course").Where("instructor_id = ?", instructorID).Order("submitted_at DESC").Find(&items).Error
	return items, err
}
