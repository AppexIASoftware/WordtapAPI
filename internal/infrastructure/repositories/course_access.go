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

const testAccessSource = "test"
const activeCourseAccess = "active"

type courseAccess struct {
	ID        string `gorm:"primaryKey"`
	UserID    string
	CourseID  string
	Status    string
	Source    string
	CreatedAt time.Time
}

func (courseAccess) TableName() string { return "course_access" }

type MySQLCourseAccessRepository struct{ db *gorm.DB }

func NewMySQLCourseAccessRepository(db *gorm.DB) *MySQLCourseAccessRepository {
	return &MySQLCourseAccessRepository{db: db}
}

func canGrantTestCourseAccess(user entities.User, course entities.Course) bool {
	return user.Role == entities.RoleStudent && user.IsActive &&
		course.OwnerType == entities.OwnerTypeTeacher && course.Status == entities.ContentStatusPublished &&
		course.AccessTier == entities.AccessTierCourse && course.PriceCents > 0
}

// Test grants are never accepted unless the caller explicitly enables them.
// Other origins must be implemented deliberately before they grant access.
func courseAccessAllowed(access courseAccess, allowTest bool) bool {
	return access.Status == activeCourseAccess && access.Source == testAccessSource && allowTest
}

func (r *MySQLCourseAccessRepository) HasTestAccess(ctx context.Context, userID, courseID string, allowTest bool) (bool, error) {
	var access courseAccess
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&access).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return courseAccessAllowed(access, allowTest), nil
}


func (r *MySQLCourseAccessRepository) GrantTestAccess(ctx context.Context, userID, courseID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user entities.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		var course entities.Course
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&course, "id = ?", courseID).Error; err != nil {
			return err
		}
		if !canGrantTestCourseAccess(user, course) {
			return commands.ErrInvalidCourseAccess
		}
		var existing courseAccess
		err := tx.Where("user_id = ? AND course_id = ?", userID, courseID).First(&existing).Error
		if err == nil {
			if existing.Status != activeCourseAccess {
				return commands.ErrInvalidCourseAccess
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&courseAccess{ID: uuid.NewString(), UserID: userID, CourseID: courseID, Status: activeCourseAccess, Source: testAccessSource, CreatedAt: time.Now().UTC()}).Error
	})
}
