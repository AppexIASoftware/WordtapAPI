package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLAdminStudentRepository implements AdminStudentRepository using GORM/MySQL.
type MySQLAdminStudentRepository struct {
	db *gorm.DB
}

// NewMySQLAdminStudentRepository creates a MySQLAdminStudentRepository.
func NewMySQLAdminStudentRepository(db *gorm.DB) *MySQLAdminStudentRepository {
	return &MySQLAdminStudentRepository{db: db}
}

// GetStudentDetail returns profile, learning stats, course progress, and classroom enrollments.
func (r *MySQLAdminStudentRepository) GetStudentDetail(ctx context.Context, studentID string) (*domainRepo.AdminStudentDetail, error) {
	var student entities.User
	err := r.db.WithContext(ctx).
		Preload("LearningStats").
		Where("id = ? AND role = ?", studentID, entities.RoleStudent).
		First(&student).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainRepo.ErrStudentNotFound
		}
		return nil, err
	}

	detail := &domainRepo.AdminStudentDetail{
		ID:            student.ID,
		Name:          student.Name,
		Email:         student.Email,
		LearningLevel: student.LearningLevel,
		AccessTier:    string(student.AccessTier),
		IsActive:      student.IsActive,
		CreatedAt:     student.CreatedAt,
		Courses:       []domainRepo.AdminStudentCourseProgress{},
		Classrooms:    []domainRepo.AdminStudentClassroom{},
	}

	if student.LearningStats != nil {
		detail.CurrentStreakDays = student.LearningStats.CurrentStreakDays
		detail.TotalPoints = student.LearningStats.TotalPoints
		detail.LessonsCompleted = student.LearningStats.LessonsCompleted
		detail.WordsLearned = student.LearningStats.WordsLearned
		detail.LastActivityDate = student.LearningStats.LastActivityDate
	}

	var courseRows []struct {
		CourseID         string
		CourseTitle      string
		CompletedLessons int
		LastActivityAt   *time.Time
	}
	r.db.WithContext(ctx).
		Table("user_lesson_progresses ulp").
		Select("c.id as course_id, c.title as course_title, COUNT(DISTINCT ulp.lesson_id) as completed_lessons, MAX(ulp.updated_at) as last_activity_at").
		Joins("INNER JOIN lessons l ON l.id = ulp.lesson_id").
		Joins("INNER JOIN courses c ON c.id = l.course_id").
		Where("ulp.user_id = ?", studentID).
		Group("c.id, c.title").
		Order("last_activity_at DESC").
		Scan(&courseRows)

	if len(courseRows) > 0 {
		courseIDs := make([]string, 0, len(courseRows))
		for _, cr := range courseRows {
			courseIDs = append(courseIDs, cr.CourseID)
		}
		totalByCourse := make(map[string]int)
		var totals []struct {
			CourseID string
			Total    int
		}
		r.db.WithContext(ctx).
			Table("lessons").
			Select("course_id, COUNT(*) as total").
			Where("course_id IN ? AND status = ?", courseIDs, entities.ContentStatusPublished).
			Group("course_id").
			Scan(&totals)
		for _, t := range totals {
			totalByCourse[t.CourseID] = t.Total
		}

		for _, cr := range courseRows {
			total := totalByCourse[cr.CourseID]
			pct := 0
			if total > 0 {
				pct = (cr.CompletedLessons * 100) / total
				if pct > 100 {
					pct = 100
				}
			}
			detail.Courses = append(detail.Courses, domainRepo.AdminStudentCourseProgress{
				CourseID:         cr.CourseID,
				CourseTitle:      cr.CourseTitle,
				CompletedLessons: cr.CompletedLessons,
				TotalLessons:     total,
				ProgressPercent:  pct,
				LastActivityAt:   cr.LastActivityAt,
			})
		}
	}

	var classrooms []domainRepo.AdminStudentClassroom
	if err := r.db.WithContext(ctx).
		Table("classroom_enrollments ce").
		Select("c.id as classroom_id, c.name as classroom_name, COALESCE(u.name, 'Docente') as teacher_name, ce.enrolled_at as enrolled_at").
		Joins("INNER JOIN classrooms c ON c.id = ce.classroom_id").
		Joins("LEFT JOIN users u ON u.id = c.teacher_id").
		Where("ce.student_id = ?", studentID).
		Order("ce.enrolled_at DESC").
		Scan(&classrooms).Error; err != nil {
		return nil, err
	}
	if classrooms != nil {
		detail.Classrooms = classrooms
	}

	return detail, nil
}

// UpdateStudentStatus toggles a student's active status and revokes sessions when suspended.
func (r *MySQLAdminStudentRepository) UpdateStudentStatus(ctx context.Context, studentID string, isActive bool) (*domainRepo.AdminStudentStatusResult, error) {
	var student entities.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND role = ?", studentID, entities.RoleStudent).
		First(&student).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainRepo.ErrStudentNotFound
		}
		return nil, err
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.User{}).
			Where("id = ? AND role = ?", studentID, entities.RoleStudent).
			Update("is_active", isActive).Error; err != nil {
			return err
		}
		if !isActive {
			if err := tx.Where("user_id = ?", studentID).Delete(&entities.UserSession{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &domainRepo.AdminStudentStatusResult{
		ID:       student.ID,
		Name:     student.Name,
		Email:    student.Email,
		IsActive: isActive,
	}, nil
}
