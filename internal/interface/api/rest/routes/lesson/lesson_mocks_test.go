package lesson_test

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v5"

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/lesson"
)

type mockLessonRepository struct {
	lessons []entities.Lesson
	items   []entities.LessonItem
}

func (m *mockLessonRepository) FindByIDOrSlug(ctx context.Context, identifier string) (*entities.Lesson, error) {
	for _, l := range m.lessons {
		if l.ID == identifier || l.Slug == identifier {
			return &l, nil
		}
	}
	return nil, errors.New("lesson not found")
}

func (m *mockLessonRepository) FindByCourseID(ctx context.Context, courseID string) ([]entities.Lesson, error) {
	var res []entities.Lesson
	for _, l := range m.lessons {
		if l.CourseID == courseID {
			res = append(res, l)
		}
	}
	return res, nil
}

func (m *mockLessonRepository) FindByID(ctx context.Context, id string) (*entities.Lesson, error) {
	for _, l := range m.lessons {
		if l.ID == id {
			return &l, nil
		}
	}
	return nil, errors.New("lesson not found")
}

func (m *mockLessonRepository) Create(ctx context.Context, l *entities.Lesson) error {
	m.lessons = append(m.lessons, *l)
	return nil
}

func (m *mockLessonRepository) Update(ctx context.Context, l *entities.Lesson) error {
	for i, existing := range m.lessons {
		if existing.ID == l.ID {
			m.lessons[i] = *l
			return nil
		}
	}
	return errors.New("lesson not found")
}

func (m *mockLessonRepository) Delete(ctx context.Context, id string) error {
	var remaining []entities.Lesson
	for _, l := range m.lessons {
		if l.ID != id {
			remaining = append(remaining, l)
		}
	}
	m.lessons = remaining
	return nil
}

func (m *mockLessonRepository) SaveItem(ctx context.Context, item *entities.LessonItem) error {
	for i, existing := range m.items {
		if existing.ID == item.ID {
			if existing.LessonID != item.LessonID {
				return apperrors.New("La tarjeta pertenece a otra lección.")
			}
			m.items[i] = *item
			return nil
		}
	}
	m.items = append(m.items, *item)
	return nil
}

func (m *mockLessonRepository) DeleteItem(ctx context.Context, lessonID, itemID string) error {
	var remaining []entities.LessonItem
	deleted := false
	for _, it := range m.items {
		if it.ID == itemID && it.LessonID == lessonID {
			deleted = true
			continue
		}
		remaining = append(remaining, it)
	}
	if !deleted {
		return apperrors.New("No se encontró la tarjeta en esta lección.")
	}
	m.items = remaining
	return nil
}

type mockCourseRepoForLesson struct {
	courses []entities.Course
}

func (m *mockCourseRepoForLesson) Create(ctx context.Context, course *entities.Course) error {
	m.courses = append(m.courses, *course)
	return nil
}
func (m *mockCourseRepoForLesson) FindByAuthor(context.Context, string) ([]entities.Course, error) {
	return nil, nil
}
func (m *mockCourseRepoForLesson) FindPublished(context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (m *mockCourseRepoForLesson) FindAll(context.Context) ([]entities.Course, error) {
	return m.courses, nil
}
func (m *mockCourseRepoForLesson) FindAllByStatus(_ context.Context, status entities.ContentStatus, ownerType entities.OwnerType) ([]entities.Course, error) {
	var result []entities.Course
	all, _ := m.FindAll(context.Background())
	for _, c := range all {
		if c.Status == status && (ownerType == "" || c.OwnerType == ownerType) {
			result = append(result, c)
		}
	}
	return result, nil
}
func (m *mockCourseRepoForLesson) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	for _, c := range m.courses {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, errors.New("course not found")
}
func (m *mockCourseRepoForLesson) Update(context.Context, *entities.Course) error { return nil }
func (m *mockCourseRepoForLesson) Delete(context.Context, string) error           { return nil }

func setupLessonTestServer() (*echo.Echo, *security.JWTService, *mockLessonRepository, *mockCourseRepoForLesson) {
	e := echo.New()
	jwtSvc := security.NewJWTService("test-secret-key-32-bytes-minimum!", 15*time.Minute, time.Hour)
	lessonRepo := &mockLessonRepository{}
	courseRepo := &mockCourseRepoForLesson{}

	listLessonsHandler := lessonQuery.NewListCourseLessonsHandler(lessonRepo, courseRepo)
	createLessonHandler := lessonCmd.NewCreateLessonHandler(lessonRepo, courseRepo)
	updateLessonHandler := lessonCmd.NewUpdateLessonHandler(lessonRepo, courseRepo)
	deleteLessonHandler := lessonCmd.NewDeleteLessonHandler(lessonRepo, courseRepo)
	saveLessonItemHandler := lessonCmd.NewSaveLessonItemHandler(lessonRepo, courseRepo)
	deleteLessonItemHandler := lessonCmd.NewDeleteLessonItemHandler(lessonRepo, courseRepo)

	router := lesson.NewLessonRouter(
		lessonRepo,
		listLessonsHandler,
		createLessonHandler,
		updateLessonHandler,
		deleteLessonHandler,
		saveLessonItemHandler,
		deleteLessonItemHandler,
	)

	authRequired := restMiddleware.RequireAuth(jwtSvc)
	studentOrStaff := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin, entities.RoleModerator, entities.RoleStudent)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)

	v1 := e.Group("/api/v1")
	router.RegisterRoutes(v1, authRequired, studentOrStaff, instructorOrAdmin)

	return e, jwtSvc, lessonRepo, courseRepo
}
