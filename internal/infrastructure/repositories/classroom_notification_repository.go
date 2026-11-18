package repositories

import (
	"context"
	"strings"
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
	r.syncReadCounts(ctx, notifications)
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
	r.syncReadCounts(ctx, notifications)
	return notifications, nil
}

// ListAll lists notifications with flexible filters (admin global scope).
func (r *MySQLClassroomNotificationRepository) ListAll(ctx context.Context, filter domainRepo.ClassroomNotificationFilter, limit, offset int) ([]entities.ClassroomNotification, error) {
	notifications := make([]entities.ClassroomNotification, 0)
	query := applyAdminNotificationFilter(r.db.WithContext(ctx).
		Preload("Classroom").
		Preload("Teacher").
		Preload("TargetCourse").
		Preload("TargetLesson").
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset), filter)

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

	ids := make([]string, 0, len(notifs))
	idSet := make(map[string]struct{}, len(notifs))
	for _, n := range notifs {
		if _, exists := idSet[n.ID]; !exists && n.ID != "" {
			idSet[n.ID] = struct{}{}
			ids = append(ids, n.ID)
		}
	}
	if len(ids) == 0 {
		return
	}

	// Single-roundtrip FK aggregation replacing heuristic title grouping
	type NotifReadStat struct {
		ClassroomNotificationID string
		Total                   int
	}
	var stats []NotifReadStat
	err := r.db.WithContext(ctx).Model(&entities.Notification{}).
		Select("classroom_notification_id, count(*) as total").
		Where("classroom_notification_id IN ? AND is_read = ?", ids, true).
		Group("classroom_notification_id").
		Scan(&stats).Error
	if err != nil {
		return
	}

	countsByID := make(map[string]int, len(stats))
	for _, s := range stats {
		countsByID[s.ClassroomNotificationID] = s.Total
	}

	for i := range notifs {
		if readCount := countsByID[notifs[i].ID]; readCount > notifs[i].ReadCount {
			notifs[i].ReadCount = readCount
		}
		if notifs[i].ReadCount > 0 && notifs[i].Status != entities.ClassroomNotificationStatusSent {
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
	query := applyAdminNotificationFilter(
		r.db.WithContext(ctx).Model(&entities.ClassroomNotification{}),
		filter,
	)

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func applyAdminNotificationFilter(query *gorm.DB, filter domainRepo.ClassroomNotificationFilter) *gorm.DB {
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
	if filter.DateFrom != nil {
		if parsed, ok := parseFilterTime(*filter.DateFrom); ok {
			query = query.Where("sent_at >= ?", parsed)
		}
	}
	if filter.DateTo != nil {
		if parsed, ok := parseFilterTime(*filter.DateTo); ok {
			query = query.Where("sent_at <= ?", parsed)
		}
	}
	return query
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
	if len(filters) == 0 {
		return query
	}
	f := filters[0]
	if f.Status != nil {
		query = query.Where("status = ?", *f.Status)
	}
	if f.DateFrom != nil {
		if parsed, ok := parseFilterTime(*f.DateFrom); ok {
			query = query.Where("sent_at >= ?", parsed)
		}
	}
	if f.DateTo != nil {
		if parsed, ok := parseFilterTime(*f.DateTo); ok {
			query = query.Where("sent_at <= ?", parsed)
		}
	}
	return query
}

func parseFilterTime(raw string) (time.Time, bool) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, val); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}
