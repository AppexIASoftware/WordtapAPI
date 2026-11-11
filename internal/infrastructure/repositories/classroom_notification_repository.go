package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLClassroomNotificationRepository implements ClassroomNotificationRepository using GORM and MySQL.
type MySQLClassroomNotificationRepository struct {
	db *gorm.DB
}

// NewMySQLClassroomNotificationRepository creates a MySQL repository instance for classroom notifications.
func NewMySQLClassroomNotificationRepository(db *gorm.DB) domainRepo.ClassroomNotificationRepository {
	return &MySQLClassroomNotificationRepository{db: db}
}

// Create inserts a new classroom notification record.
func (r *MySQLClassroomNotificationRepository) Create(ctx context.Context, notification *entities.ClassroomNotification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

// ListByClassroom lists notifications for a specific classroom (teacher scope).
func (r *MySQLClassroomNotificationRepository) ListByClassroom(ctx context.Context, classroomID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	notifications := make([]entities.ClassroomNotification, 0)
	err := r.db.WithContext(ctx).
		Where("classroom_id = ?", classroomID).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetCourse").
		Preload("TargetLesson").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	r.syncReadCounts(ctx, notifications)
	return notifications, nil
}

// ListByTeacher lists notifications sent by a specific teacher.
func (r *MySQLClassroomNotificationRepository) ListByTeacher(ctx context.Context, teacherID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	notifications := make([]entities.ClassroomNotification, 0)
	err := r.db.WithContext(ctx).
		Where("teacher_id = ?", teacherID).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetCourse").
		Preload("TargetLesson").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	r.syncReadCounts(ctx, notifications)
	return notifications, nil
}

// ListByCourse lists notifications targeting a specific course (admin scope).
func (r *MySQLClassroomNotificationRepository) ListByCourse(ctx context.Context, courseID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	notifications := make([]entities.ClassroomNotification, 0)
	err := r.db.WithContext(ctx).
		Where("target_course_id = ?", courseID).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetLesson").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

// ListByLesson lists notifications targeting a specific lesson (admin scope).
func (r *MySQLClassroomNotificationRepository) ListByLesson(ctx context.Context, lessonID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	notifications := make([]entities.ClassroomNotification, 0)
	err := r.db.WithContext(ctx).
		Where("target_lesson_id = ?", lessonID).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetCourse").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

// ListAll lists notifications with flexible filters (admin global scope).
func (r *MySQLClassroomNotificationRepository) ListAll(ctx context.Context, filter domainRepo.ClassroomNotificationFilter, limit, offset int) ([]entities.ClassroomNotification, error) {
	notifications := make([]entities.ClassroomNotification, 0)
	query := r.db.WithContext(ctx).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetCourse").
		Preload("TargetLesson").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset)

	if filter.ClassroomID != nil && *filter.ClassroomID != "" {
		query = query.Where("classroom_id = ?", *filter.ClassroomID)
	}
	if filter.TeacherID != nil && *filter.TeacherID != "" {
		query = query.Where("teacher_id = ?", *filter.TeacherID)
	}
	if filter.CourseID != nil && *filter.CourseID != "" {
		query = query.Where("target_course_id = ?", *filter.CourseID)
	}
	if filter.LessonID != nil && *filter.LessonID != "" {
		query = query.Where("target_lesson_id = ?", *filter.LessonID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.DateFrom != nil && *filter.DateFrom != "" {
		query = query.Where("sent_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil && *filter.DateTo != "" {
		query = query.Where("sent_at <= ?", *filter.DateTo)
	}

	err := query.Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	r.syncReadCounts(ctx, notifications)
	return notifications, nil
}

func (r *MySQLClassroomNotificationRepository) syncReadCounts(ctx context.Context, notifs []entities.ClassroomNotification) {
	if len(notifs) == 0 {
		return
	}

	// 1. Collect unique titles for single-roundtrip aggregation
	titles := make([]string, 0, len(notifs))
	titleSet := make(map[string]struct{}, len(notifs))
	for _, n := range notifs {
		if _, exists := titleSet[n.Title]; !exists && n.Title != "" {
			titleSet[n.Title] = struct{}{}
			titles = append(titles, n.Title)
		}
	}
	if len(titles) == 0 {
		return
	}

	// 2. Aggregated batch query: single roundtrip avoiding N+1 roundtrips
	type TitleReadStat struct {
		Title string
		Total int
	}
	var stats []TitleReadStat
	err := r.db.WithContext(ctx).Model(&entities.Notification{}).
		Select("title, count(*) as total").
		Where("title IN ? AND is_read = ?", titles, true).
		Group("title").
		Scan(&stats).Error
	if err != nil {
		return
	}

	countsByTitle := make(map[string]int, len(stats))
	for _, s := range stats {
		countsByTitle[s.Title] = s.Total
	}

	// 3. In-memory metric mapping for fast reads without locking
	for i := range notifs {
		readCount := countsByTitle[notifs[i].Title]
		notifs[i].ReadCount = readCount
		if readCount > 0 && notifs[i].Status != entities.ClassroomNotificationStatusSent {
			notifs[i].Status = entities.ClassroomNotificationStatusSent
		} else if notifs[i].Status == entities.ClassroomNotificationStatusFailed && notifs[i].RecipientCount > 0 {
			notifs[i].Status = entities.ClassroomNotificationStatusPartial
		}
	}
}

// CountByClassroom counts notifications for a specific classroom.
func (r *MySQLClassroomNotificationRepository) CountByClassroom(ctx context.Context, classroomID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.ClassroomNotification{}).
		Where("classroom_id = ?", classroomID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountByTeacher counts notifications sent by a specific teacher.
func (r *MySQLClassroomNotificationRepository) CountByTeacher(ctx context.Context, teacherID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.ClassroomNotification{}).
		Where("teacher_id = ?", teacherID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountAll counts notifications with flexible filters (admin global scope).
func (r *MySQLClassroomNotificationRepository) CountAll(ctx context.Context, filter domainRepo.ClassroomNotificationFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&entities.ClassroomNotification{})

	if filter.ClassroomID != nil && *filter.ClassroomID != "" {
		query = query.Where("classroom_id = ?", *filter.ClassroomID)
	}
	if filter.TeacherID != nil && *filter.TeacherID != "" {
		query = query.Where("teacher_id = ?", *filter.TeacherID)
	}
	if filter.CourseID != nil && *filter.CourseID != "" {
		query = query.Where("target_course_id = ?", *filter.CourseID)
	}
	if filter.LessonID != nil && *filter.LessonID != "" {
		query = query.Where("target_lesson_id = ?", *filter.LessonID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.DateFrom != nil && *filter.DateFrom != "" {
		query = query.Where("sent_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil && *filter.DateTo != "" {
		query = query.Where("sent_at <= ?", *filter.DateTo)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ListClassroomCoursesForTeacher returns the course IDs stored on the classroom.
func (r *MySQLClassroomNotificationRepository) ListClassroomCoursesForTeacher(ctx context.Context, classroomID string) ([]string, error) {
	var classroom entities.Classroom
	err := r.db.WithContext(ctx).Select("id", "course_id", "assigned_courses").First(&classroom, "id = ?", classroomID).Error
	if err != nil {
		return nil, err
	}
	return assignedCourseIDs(classroom), nil
}

// Helper to build scope for teacher: classroom_id + target_course_id IN (courses of classroom)
func (r *MySQLClassroomNotificationRepository) ListByClassroomWithCourseScope(ctx context.Context, classroomID string, limit, offset int, filters ...domainRepo.ClassroomNotificationFilter) ([]entities.ClassroomNotification, error) {
	// Query course IDs associated with classroom
	courseIDs, err := r.ListClassroomCoursesForTeacher(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	notifications := make([]entities.ClassroomNotification, 0)
	query := r.db.WithContext(ctx).
		Where("classroom_id = ?", classroomID).
		Where("(target_course_id IS NULL OR target_course_id IN ?)", courseIDs).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetCourse").
		Preload("TargetLesson").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset)

	err = applyTeacherHistoryFilters(query, filters).Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	r.syncReadCounts(ctx, notifications)
	return notifications, nil
}

// CountByClassroomWithCourseScope counts classroom notifications within course scope.
func (r *MySQLClassroomNotificationRepository) CountByClassroomWithCourseScope(ctx context.Context, classroomID string, filters ...domainRepo.ClassroomNotificationFilter) (int64, error) {
	courseIDs, err := r.ListClassroomCoursesForTeacher(ctx, classroomID)
	if err != nil {
		return 0, err
	}
	var count int64
	query := r.db.WithContext(ctx).
		Model(&entities.ClassroomNotification{}).
		Where("classroom_id = ?", classroomID).
		Where("(target_course_id IS NULL OR target_course_id IN ?)", courseIDs)
	err = applyTeacherHistoryFilters(query, filters).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func applyTeacherHistoryFilters(query *gorm.DB, filters []domainRepo.ClassroomNotificationFilter) *gorm.DB {
	if len(filters) == 0 { return query }
	f := filters[0]
	if f.Status != nil { query = query.Where("status = ?", *f.Status) }
	if f.DateFrom != nil { if value, err := time.Parse(time.RFC3339Nano, *f.DateFrom); err == nil { query = query.Where("sent_at >= ?", value.UTC()) } }
	if f.DateTo != nil { if value, err := time.Parse(time.RFC3339Nano, *f.DateTo); err == nil { query = query.Where("sent_at <= ?", value.UTC()) } }
	return query
}
