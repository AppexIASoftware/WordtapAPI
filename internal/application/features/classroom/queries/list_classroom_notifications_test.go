package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type mockClassroomNotifRepo struct {
	notifications []entities.ClassroomNotification
	err           error
	// Mock course filtering mapping for teacher scope simulation
	classroomCourses map[string][]string
}

func (m *mockClassroomNotifRepo) Create(ctx context.Context, notification *entities.ClassroomNotification) error {
	return nil
}

func (m *mockClassroomNotifRepo) ListByClassroom(ctx context.Context, classroomID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	return m.notifications, m.err
}

func (m *mockClassroomNotifRepo) ListByTeacher(ctx context.Context, teacherID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	return m.notifications, m.err
}

func (m *mockClassroomNotifRepo) ListByCourse(ctx context.Context, courseID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	return m.notifications, m.err
}

func (m *mockClassroomNotifRepo) ListByLesson(ctx context.Context, lessonID string, limit, offset int) ([]entities.ClassroomNotification, error) {
	return m.notifications, m.err
}

func (m *mockClassroomNotifRepo) ListAll(ctx context.Context, filter domainRepo.ClassroomNotificationFilter, limit, offset int) ([]entities.ClassroomNotification, error) {
	return m.notifications, m.err
}

func (m *mockClassroomNotifRepo) CountByClassroom(ctx context.Context, classroomID string) (int64, error) {
	return int64(len(m.notifications)), m.err
}

func (m *mockClassroomNotifRepo) CountByTeacher(ctx context.Context, teacherID string) (int64, error) {
	return int64(len(m.notifications)), m.err
}

func (m *mockClassroomNotifRepo) CountAll(ctx context.Context, filter domainRepo.ClassroomNotificationFilter) (int64, error) {
	return int64(len(m.notifications)), m.err
}

func (m *mockClassroomNotifRepo) ListClassroomCoursesForTeacher(ctx context.Context, classroomID string) ([]string, error) {
	if m.classroomCourses != nil {
		if courses, ok := m.classroomCourses[classroomID]; ok {
			return courses, nil
		}
	}
	return []string{"course-1", "course-2"}, nil
}

func (m *mockClassroomNotifRepo) ListByClassroomWithCourseScope(ctx context.Context, classroomID string, limit, offset int, filters ...domainRepo.ClassroomNotificationFilter) ([]entities.ClassroomNotification, error) {
	courseIDs, _ := m.ListClassroomCoursesForTeacher(ctx, classroomID)
	filtered := make([]entities.ClassroomNotification, 0)
	for _, n := range m.notifications {
		if len(filters) > 0 {
			f := filters[0]
			if f.Status != nil && n.Status != *f.Status { continue }
			if f.DateFrom != nil { start, _ := time.Parse(time.RFC3339Nano, *f.DateFrom); if n.SentAt.Before(start) { continue } }
			if f.DateTo != nil { end, _ := time.Parse(time.RFC3339Nano, *f.DateTo); if n.SentAt.After(end) { continue } }
		}
		if n.TargetCourseID != nil {
			for _, cid := range courseIDs {
				if *n.TargetCourseID == cid {
					filtered = append(filtered, n)
					break
				}
			}
		} else {
			// Notificaciones sin target_course_id (open_classroom) se incluyen
			filtered = append(filtered, n)
		}
	}
	return filtered, m.err
}

func (m *mockClassroomNotifRepo) CountByClassroomWithCourseScope(ctx context.Context, classroomID string, filters ...domainRepo.ClassroomNotificationFilter) (int64, error) {
	filtered, _ := m.ListByClassroomWithCourseScope(ctx, classroomID, 100, 0, filters...)
	return int64(len(filtered)), m.err
}

type mockClassroomRepo struct {
	classroom *entities.Classroom
}

func (m *mockClassroomRepo) Create(ctx context.Context, classroom *entities.Classroom) error {
	return nil
}
func (m *mockClassroomRepo) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	if m.classroom != nil && m.classroom.ID == id {
		return m.classroom, nil
	}
	return nil, nil
}
func (m *mockClassroomRepo) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepo) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepo) EnrollStudent(ctx context.Context, enrollment *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockClassroomRepo) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepo) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepo) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockClassroomRepo) CreateStumble(ctx context.Context, stumble *entities.StudentStumble) error {
	return nil
}
func (m *mockClassroomRepo) Update(ctx context.Context, classroom *entities.Classroom) error {
	return nil
}

func TestListClassroomNotificationsHandler_TeacherScope(t *testing.T) {
	now := time.Now().UTC()
	repo := &mockClassroomNotifRepo{
		notifications: []entities.ClassroomNotification{
			{
				ID:             "notif-1",
				ClassroomID:    stringPtr("cls-1"),
				TeacherID:      "teacher-1",
				Title:          "Aviso 1",
				Message:        "Mensaje 1",
				ActionType:     "open_content",
				TargetCourseID: stringPtr("course-1"),
				SentAt:         now.Add(-time.Hour),
				RecipientCount: 20,
				ReadCount:      15,
				Status:         entities.ClassroomNotificationStatusSent,
			},
			{
				ID:             "notif-2",
				ClassroomID:    stringPtr("cls-1"),
				TeacherID:      "teacher-1",
				Title:          "Aviso 2",
				Message:        "Mensaje 2",
				ActionType:     "open_classroom",
				TargetCourseID: nil,
				SentAt:         now,
				RecipientCount: 20,
				ReadCount:      20,
				Status:         entities.ClassroomNotificationStatusSent,
			},
			{
				ID:             "notif-3",
				ClassroomID:    stringPtr("cls-1"),
				TeacherID:      "teacher-1",
				Title:          "Aviso 3 (curso no vinculado)",
				Message:        "Mensaje 3",
				ActionType:     "open_content",
				TargetCourseID: stringPtr("course-999"),
				SentAt:         now.Add(-2 * time.Hour),
				RecipientCount: 20,
				ReadCount:      5,
				Status:         entities.ClassroomNotificationStatusSent,
			},
		},
		classroomCourses: map[string][]string{
			"cls-1": {"course-1", "course-2"},
		},
	}
	classroomRepo := &mockClassroomRepo{
		classroom: &entities.Classroom{ID: "cls-1", TeacherID: "teacher-1"},
	}

	handler := queries.NewListClassroomNotificationsHandler(repo, classroomRepo)

	t.Run("scope teacher - lista notificaciones filtradas por cursos del aula", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:       queries.NotificationScopeTeacher,
			ClassroomID: "cls-1",
			TeacherID:   "teacher-1",
			Limit:       20,
			Offset:      0,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		// notif-1 (course-1 en aula) + notif-2 (open_classroom sin target) = 2
		// notif-3 (course-999 NO en aula) se filtra
		if len(res.Notifications) != 2 {
			t.Fatalf("se esperaban 2 notificaciones, obtuvidas %d", len(res.Notifications))
		}
		if res.Total != 2 {
			t.Fatalf("total esperado 2, obtenido %d", res.Total)
		}
	})

	t.Run("scope teacher - error si classroom_id vacío", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:       queries.NotificationScopeTeacher,
			ClassroomID: "",
			TeacherID:   "teacher-1",
		})
		if err == nil {
			t.Fatal("se esperaba error por classroom_id vacío")
		}
	})

	t.Run("scope teacher - error si teacher_id vacío", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:       queries.NotificationScopeTeacher,
			ClassroomID: "cls-1",
			TeacherID:   "",
		})
		if err == nil {
			t.Fatal("se esperaba error por teacher_id vacío")
		}
	})

	t.Run("scope teacher - error si aula no encontrada", func(t *testing.T) {
		classroomRepo.classroom = nil
		_, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:       queries.NotificationScopeTeacher,
			ClassroomID: "cls-999",
			TeacherID:   "teacher-1",
		})
		if err == nil {
			t.Fatal("se esperaba error por aula no encontrada")
		}
	})

	t.Run("scope teacher - error si no es owner del aula", func(t *testing.T) {
		classroomRepo.classroom = &entities.Classroom{ID: "cls-1", TeacherID: "other-teacher"}
		_, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:       queries.NotificationScopeTeacher,
			ClassroomID: "cls-1",
			TeacherID:   "teacher-1",
		})
		if err == nil {
			t.Fatal("se esperaba error por no ser owner")
		}
	})
}

func TestListClassroomNotificationsHandler_AdminScope(t *testing.T) {
	now := time.Now().UTC()
	repo := &mockClassroomNotifRepo{
		notifications: []entities.ClassroomNotification{
			{
				ID:             "notif-1",
				ClassroomID:    stringPtr("cls-1"),
				TeacherID:      "teacher-1",
				Title:          "Aviso Curso 1",
				Message:        "Msg",
				ActionType:     "open_content",
				TargetCourseID: stringPtr("course-1"),
				TargetLessonID: stringPtr("lesson-1"),
				SentAt:         now.Add(-time.Hour),
				RecipientCount: 20,
				ReadCount:      15,
				Status:         entities.ClassroomNotificationStatusSent,
			},
			{
				ID:             "notif-2",
				ClassroomID:    stringPtr("cls-2"),
				TeacherID:      "teacher-2",
				Title:          "Aviso Curso 2",
				Message:        "Msg",
				ActionType:     "open_content",
				TargetCourseID: stringPtr("course-2"),
				TargetLessonID: nil,
				SentAt:         now,
				RecipientCount: 15,
				ReadCount:      10,
				Status:         entities.ClassroomNotificationStatusSent,
			},
		},
	}

	handler := queries.NewListClassroomNotificationsHandler(repo, &mockClassroomRepo{})

	t.Run("scope admin - lista global sin filtros", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:  queries.NotificationScopeAdmin,
			Limit:  20,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res.Notifications) != 2 {
			t.Fatalf("se esperaban 2 notificaciones, obtuvidas %d", len(res.Notifications))
		}
		if res.Total != 2 {
			t.Fatalf("total esperado 2, obtenido %d", res.Total)
		}
	})

	t.Run("scope admin - filtro por course_id", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:     queries.NotificationScopeAdmin,
			CourseID:  "course-1",
			Limit:     20,
			Offset:    0,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		// The mock returns the static list for structure validation
		if len(res.Notifications) != 2 {
			t.Fatalf("se esperaban 2 notificaciones (mock no filtra), obtuvidas %d", len(res.Notifications))
		}
	})

	t.Run("scope admin - filtro por teacher_id", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:      queries.NotificationScopeAdmin,
			TeacherID:  "teacher-1",
			Limit:      20,
			Offset:     0,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res.Notifications) != 2 {
			t.Fatalf("se esperaban 2 notificaciones (mock no filtra), obtuvidas %d", len(res.Notifications))
		}
	})

	t.Run("scope admin - filtro por classroom_id", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:       queries.NotificationScopeAdmin,
			ClassroomID:  "cls-1",
			Limit:        20,
			Offset:       0,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res.Notifications) != 2 {
			t.Fatalf("se esperaban 2 notificaciones (mock no filtra), obtuvidas %d", len(res.Notifications))
		}
	})

	t.Run("scope admin - filtro por fecha", func(t *testing.T) {
		from := now.Add(-3 * time.Hour)
		to := now.Add(time.Hour)
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:    queries.NotificationScopeAdmin,
			DateFrom: &from,
			DateTo:   &to,
			Limit:    20,
			Offset:   0,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res.Notifications) != 2 {
			t.Fatalf("se esperaban 2 notificaciones, obtuvidas %d", len(res.Notifications))
		}
	})
}

func TestListClassroomNotificationsHandler_Defaults(t *testing.T) {
	repo := &mockClassroomNotifRepo{
		notifications: []entities.ClassroomNotification{},
	}
	handler := queries.NewListClassroomNotificationsHandler(repo, &mockClassroomRepo{})

	t.Run("defaults limit y offset", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope: queries.NotificationScopeAdmin,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if res.Limit != 20 {
			t.Fatalf("limit default esperado 20, obtenido %d", res.Limit)
		}
		if res.Offset != 0 {
			t.Fatalf("offset default esperado 0, obtenido %d", res.Offset)
		}
	})

	t.Run("clamp limit a 100 máximo", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListClassroomNotificationsQuery{
			Scope:  queries.NotificationScopeAdmin,
			Limit:  200,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if res.Limit != 100 {
			t.Fatalf("limit clamp esperado 100, obtenido %d", res.Limit)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}

func TestTeacherHistoryFiltersStatusAndDate(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	status := entities.ClassroomNotificationStatusFailed
	repo := &mockClassroomNotifRepo{notifications: []entities.ClassroomNotification{
		{ID: "match", ClassroomID: stringPtr("class"), Status: status, SentAt: start.Add(time.Hour)},
		{ID: "wrong-status", ClassroomID: stringPtr("class"), Status: entities.ClassroomNotificationStatusSent, SentAt: start.Add(time.Hour)},
		{ID: "outside-date", ClassroomID: stringPtr("class"), Status: status, SentAt: start.Add(-time.Hour)},
	}}
	h := queries.NewListClassroomNotificationsHandler(repo, &mockClassroomRepo{classroom: &entities.Classroom{ID: "class", TeacherID: "teacher"}})
	result, err := h.Handle(context.Background(), queries.ListClassroomNotificationsQuery{Scope: queries.NotificationScopeTeacher, ClassroomID: "class", TeacherID: "teacher", Status: &status, DateFrom: &start, DateTo: &end})
	if err != nil || len(result.Notifications) != 1 || result.Notifications[0].ID != "match" || result.Total != 1 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
