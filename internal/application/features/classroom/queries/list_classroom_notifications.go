package queries

import (
	"context"
	"errors"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// NotificationScope defines the query scope.
type NotificationScope string

const (
	NotificationScopeTeacher NotificationScope = "teacher" // Teacher: strictly classroom-scoped, filtered by classroom courses
	NotificationScopeAdmin   NotificationScope = "admin"   // Admin: global catalog-wide scope with filters
)

// ListClassroomNotificationsQuery contains query parameters for listing notifications.
type ListClassroomNotificationsQuery struct {
	Scope         NotificationScope
	ClassroomID   string // Required for teacher scope
	TeacherID     string // Teacher ownership validation or admin optional filter
	CourseID      string // Course filter
	LessonID      string // Lesson filter
	Status        *entities.ClassroomNotificationStatus
	DateFrom      *time.Time
	DateTo        *time.Time
	Limit         int
	Offset        int
}

type ListClassroomNotificationsResult struct {
	Notifications []entities.ClassroomNotification
	Total         int64
	Limit         int
	Offset        int
}

type ListClassroomNotificationsHandler struct {
	notificationRepo domainRepo.ClassroomNotificationRepository
	classroomRepo    domainRepo.ClassroomRepository
}

func NewListClassroomNotificationsHandler(
	notificationRepo domainRepo.ClassroomNotificationRepository,
	classroomRepo domainRepo.ClassroomRepository,
) *ListClassroomNotificationsHandler {
	return &ListClassroomNotificationsHandler{
		notificationRepo: notificationRepo,
		classroomRepo:    classroomRepo,
	}
}

func (h *ListClassroomNotificationsHandler) Handle(ctx context.Context, query ListClassroomNotificationsQuery) (*ListClassroomNotificationsResult, error) {
	// Basic validation
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	switch query.Scope {
	case NotificationScopeTeacher:
		return h.handleTeacherScope(ctx, query)
	case NotificationScopeAdmin:
		return h.handleAdminScope(ctx, query)
	default:
		return nil, errors.New("alcance de consulta inválido")
	}
}

func (h *ListClassroomNotificationsHandler) handleTeacherScope(ctx context.Context, query ListClassroomNotificationsQuery) (*ListClassroomNotificationsResult, error) {
	if query.ClassroomID == "" {
		return nil, errors.New("classroom_id es obligatorio para scope teacher")
	}
	if query.TeacherID == "" {
		return nil, errors.New("teacher_id es obligatorio para scope teacher")
	}

	// Verificar ownership del aula
	classroom, err := h.classroomRepo.FindByID(ctx, query.ClassroomID)
	if err != nil {
		return nil, err
	}
	if classroom == nil {
		return nil, errors.New("aula no encontrada")
	}
	if classroom.TeacherID != query.TeacherID {
		return nil, errors.New("no tienes permisos para ver el historial de esta aula")
	}

	filter := domainRepo.ClassroomNotificationFilter{Status: query.Status}
	if query.DateFrom != nil {
		value := query.DateFrom.UTC().Format(time.RFC3339Nano)
		filter.DateFrom = &value
	}
	if query.DateTo != nil {
		value := query.DateTo.UTC().Format(time.RFC3339Nano)
		filter.DateTo = &value
	}
	// Apply the same filters to both the page and its total.
	notifications, err := h.notificationRepo.ListByClassroomWithCourseScope(ctx, query.ClassroomID, query.Limit, query.Offset, filter)
	if err != nil {
		return nil, err
	}

	total, err := h.notificationRepo.CountByClassroomWithCourseScope(ctx, query.ClassroomID, filter)
	if err != nil {
		return nil, err
	}

	return &ListClassroomNotificationsResult{
		Notifications: notifications,
		Total:         total,
		Limit:         query.Limit,
		Offset:        query.Offset,
	}, nil
}

func (h *ListClassroomNotificationsHandler) handleAdminScope(ctx context.Context, query ListClassroomNotificationsQuery) (*ListClassroomNotificationsResult, error) {
	filter := domainRepo.ClassroomNotificationFilter{
		ClassroomID: &query.ClassroomID,
		TeacherID:   &query.TeacherID,
		CourseID:    &query.CourseID,
		LessonID:    &query.LessonID,
		Status:      query.Status,
	}

	// Clear empty filter pointers
	if query.ClassroomID == "" {
		filter.ClassroomID = nil
	}
	if query.TeacherID == "" {
		filter.TeacherID = nil
	}
	if query.CourseID == "" {
		filter.CourseID = nil
	}
	if query.LessonID == "" {
		filter.LessonID = nil
	}

	// Format DateFrom/DateTo to RFC3339 string
	if query.DateFrom != nil {
		s := query.DateFrom.Format(time.RFC3339)
		filter.DateFrom = &s
	}
	if query.DateTo != nil {
		s := query.DateTo.Format(time.RFC3339)
		filter.DateTo = &s
	}

	notifications, err := h.notificationRepo.ListAll(ctx, filter, query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	total, err := h.notificationRepo.CountAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListClassroomNotificationsResult{
		Notifications: notifications,
		Total:         total,
		Limit:         query.Limit,
		Offset:        query.Offset,
	}, nil
}
