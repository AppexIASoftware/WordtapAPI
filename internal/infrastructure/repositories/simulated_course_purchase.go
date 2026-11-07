package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type simulatedPurchase struct {
	ID          string `gorm:"primaryKey"`
	UserID      string
	CourseID    string
	AmountCents int
	Currency    string
	CreatedAt   time.Time
}

func (simulatedPurchase) TableName() string { return "simulated_course_purchases" }

type simulatedPurchaseKey struct {
	UserID         string `gorm:"primaryKey"`
	IdempotencyKey string `gorm:"primaryKey"`
	PurchaseID     string
}

func (simulatedPurchaseKey) TableName() string { return "simulated_course_purchase_keys" }

type MySQLSimulatedPurchaseRepository struct{ db *gorm.DB }

func NewMySQLSimulatedPurchaseRepository(db *gorm.DB) *MySQLSimulatedPurchaseRepository {
	return &MySQLSimulatedPurchaseRepository{db: db}
}

func purchaseDTO(p simulatedPurchase) *commands.SimulatedCoursePurchase {
	return &commands.SimulatedCoursePurchase{ID: p.ID, UserID: p.UserID, CourseID: p.CourseID, AmountCents: p.AmountCents, Currency: p.Currency, CreatedAt: p.CreatedAt}
}

func (r *MySQLSimulatedPurchaseRepository) Find(ctx context.Context, userID, courseID string) (*commands.SimulatedCoursePurchase, error) {
	var p simulatedPurchase
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return purchaseDTO(p), nil
}

func (r *MySQLSimulatedPurchaseRepository) HasPurchase(ctx context.Context, userID, courseID string) (bool, error) {
	purchase, err := r.Find(ctx, userID, courseID)
	return purchase != nil, err
}

func (r *MySQLSimulatedPurchaseRepository) Purchase(ctx context.Context, userID, courseID, key string) (*commands.SimulatedCoursePurchase, error) {
	var purchase simulatedPurchase
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize purchases for the account, including distinct retry keys.
		var user entities.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "role", "is_active").First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		if user.Role != entities.RoleStudent || !user.IsActive {
			return commands.ErrInvalidPurchase
		}
		var existingKey simulatedPurchaseKey
		err := tx.Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existingKey).Error
		if err == nil {
			if err := tx.First(&purchase, "id = ?", existingKey.PurchaseID).Error; err != nil {
				return err
			}
			if purchase.CourseID != courseID {
				return commands.ErrIdempotencyConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		err = tx.Where("user_id = ? AND course_id = ?", userID, courseID).First(&purchase).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var course entities.Course
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&course, "id = ?", courseID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return commands.ErrCourseNotPurchasable
				}
				return err
			}
			if course.OwnerType != entities.OwnerTypeTeacher || course.Status != entities.ContentStatusPublished || course.AccessTier != entities.AccessTierCourse || course.PriceCents <= 0 {
				return commands.ErrCourseNotPurchasable
			}
			purchase = simulatedPurchase{ID: uuid.NewString(), UserID: userID, CourseID: courseID, AmountCents: course.PriceCents, Currency: commands.TeacherCourseCurrency, CreatedAt: time.Now().UTC()}
			if err := tx.Create(&purchase).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return tx.Create(&simulatedPurchaseKey{UserID: userID, IdempotencyKey: key, PurchaseID: purchase.ID}).Error
	})
	if err != nil {
		return nil, err
	}
	return purchaseDTO(purchase), nil
}
