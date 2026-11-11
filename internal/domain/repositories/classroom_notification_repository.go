package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// ClassroomNotificationRepository defines persistence operations for classroom notification history.
type ClassroomNotificationRepository interface {
	Create(ctx context.Context, notification *entities.ClassroomNotification) error
	ListByClassroom(ctx context.Context, classroomID string, limit, offset int) ([]entities.ClassroomNotification, error)
	ListByTeacher(ctx context.Context, teacherID string, limit, offset int) ([]entities.ClassroomNotification, error)
	ListByCourse(ctx context.Context, courseID string, limit, offset int) ([]entities.ClassroomNotification, error)
	ListByLesson(ctx context.Context, lessonID string, limit, offset int) ([]entities.ClassroomNotification, error)
	ListAll(ctx context.Context, filter ClassroomNotificationFilter, limit, offset int) ([]entities.ClassroomNotification, error)
	CountByClassroom(ctx context.Context, classroomID string) (int64, error)
	CountByTeacher(ctx context.Context, teacherID string) (int64, error)
	CountAll(ctx context.Context, filter ClassroomNotificationFilter) (int64, error)
	// Teacher scope: classroom + course filter (courses associated with the classroom)
	ListClassroomCoursesForTeacher(ctx context.Context, classroomID string) ([]string, error)
	ListByClassroomWithCourseScope(ctx context.Context, classroomID string, limit, offset int, filters ...ClassroomNotificationFilter) ([]entities.ClassroomNotification, error)
	CountByClassroomWithCourseScope(ctx context.Context, classroomID string, filters ...ClassroomNotificationFilter) (int64, error)
}

// ClassroomNotificationFilter contains query filters for notification search and audit.
type ClassroomNotificationFilter struct {
	ClassroomID *string
	TeacherID   *string
	CourseID    *string
	LessonID    *string
	Status      *entities.ClassroomNotificationStatus
	DateFrom    *string // RFC3339
	DateTo      *string // RFC3339
}
