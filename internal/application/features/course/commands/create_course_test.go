package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockCourseRepo struct {
	createdCourse *entities.Course
	createErr     error
}

func (m *mockCourseRepo) Create(ctx context.Context, course *entities.Course) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdCourse = course
	return nil
}

func (m *mockCourseRepo) FindByAuthor(ctx context.Context, authorID string) ([]entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepo) FindPublished(ctx context.Context) ([]entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepo) FindAll(ctx context.Context) ([]entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepo) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepo) Update(ctx context.Context, course *entities.Course) error {
	return nil
}

func (m *mockCourseRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCreateCourseHandler(t *testing.T) {
	repo := &mockCourseRepo{}
	handler := commands.NewCreateCourseHandler(repo)

	t.Run("success create course as draft", func(t *testing.T) {
		cmd := commands.CreateCourseCommand{
			Title:      "Inglés para Viajeros",
			Level:      "A2",
			AccessTier: entities.AccessTierFree,
			AuthorID:   "usr-teacher-1",
		}

		course, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if course == nil {
			t.Fatal("expected course to not be nil")
		}
		if course.Title != "Inglés para Viajeros" {
			t.Errorf("expected title 'Inglés para Viajeros', got '%s'", course.Title)
		}
		if course.Status != entities.ContentStatusDraft {
			t.Errorf("expected status 'draft', got '%s'", course.Status)
		}
		if course.CreatedBy == nil || *course.CreatedBy != "usr-teacher-1" {
			t.Errorf("expected author 'usr-teacher-1', got %v", course.CreatedBy)
		}
		if course.OwnerType != entities.OwnerTypeTeacher {
			t.Errorf("expected teacher ownership, got '%s'", course.OwnerType)
		}
		if course.Slug == "" {
			t.Error("expected non-empty slug")
		}
	})

	t.Run("admin course keeps platform ownership", func(t *testing.T) {
		cmd := commands.CreateCourseCommand{
			Title:     "Curso de Plataforma",
			Level:     "B1",
			AuthorID:  "usr-admin-1",
			OwnerType: entities.OwnerTypePlatform,
		}

		course, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if course.OwnerType != entities.OwnerTypePlatform {
			t.Errorf("expected platform ownership, got '%s'", course.OwnerType)
		}
	})

	t.Run("missing title fails validation", func(t *testing.T) {
		cmd := commands.CreateCourseCommand{
			Title:    "   ",
			AuthorID: "usr-teacher-1",
		}

		_, err := handler.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error on empty title, got nil")
		}
	})

	t.Run("missing author id fails validation", func(t *testing.T) {
		cmd := commands.CreateCourseCommand{
			Title:    "Valid Title",
			AuthorID: "",
		}

		_, err := handler.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error on empty author id, got nil")
		}
	})

	t.Run("teacher course with price below 2000 cents fails validation", func(t *testing.T) {
		cmd := commands.CreateCourseCommand{
			Title:      "Curso Docente Barato",
			Level:      "B1",
			AccessTier: entities.AccessTierCourse,
			PriceCents: 1500,
			AuthorID:   "usr-teacher-1",
			OwnerType:  entities.OwnerTypeTeacher,
		}

		_, err := handler.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error for price below S/ 20.00, got nil")
		}
	})

	t.Run("teacher course with price 2000 cents succeeds", func(t *testing.T) {
		cmd := commands.CreateCourseCommand{
			Title:      "Curso Docente Precio Minimo",
			Level:      "B1",
			AccessTier: entities.AccessTierCourse,
			PriceCents: 2000,
			AuthorID:   "usr-teacher-1",
			OwnerType:  entities.OwnerTypeTeacher,
		}

		course, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if course.PriceCents != 2000 {
			t.Errorf("expected price 2000, got %d", course.PriceCents)
		}
	})

	t.Run("teacher course with dynamic min price setting enforces configured minimum", func(t *testing.T) {
		mockSettings := &mockSettingRepo{
			settings: map[string]string{
				commands.CourseMinPriceSettingKey: "2500",
			},
		}
		handlerWithSettings := commands.NewCreateCourseHandler(repo, mockSettings)

		// 2000 is now below the 2500 setting
		_, err := handlerWithSettings.Handle(context.Background(), commands.CreateCourseCommand{
			Title:      "Curso Docente 2000 vs 2500",
			Level:      "B1",
			AccessTier: entities.AccessTierCourse,
			PriceCents: 2000,
			AuthorID:   "usr-teacher-1",
			OwnerType:  entities.OwnerTypeTeacher,
		})
		if err == nil {
			t.Fatal("expected error when price is below configured 2500 setting, got nil")
		}

		// 2500 meets the configured minimum
		course, err := handlerWithSettings.Handle(context.Background(), commands.CreateCourseCommand{
			Title:      "Curso Docente 2500 OK",
			Level:      "B1",
			AccessTier: entities.AccessTierCourse,
			PriceCents: 2500,
			AuthorID:   "usr-teacher-1",
			OwnerType:  entities.OwnerTypeTeacher,
		})
		if err != nil {
			t.Fatalf("unexpected error for 2500 price: %v", err)
		}
		if course.PriceCents != 2500 {
			t.Errorf("expected price 2500, got %d", course.PriceCents)
		}
	})
}

type mockSettingRepo struct {
	settings map[string]string
}

func (m *mockSettingRepo) Get(_ context.Context, key string) (*entities.PlatformSetting, error) {
	if m.settings != nil {
		if v, ok := m.settings[key]; ok {
			return &entities.PlatformSetting{Key: key, Value: v}, nil
		}
	}
	return nil, nil
}

func (m *mockSettingRepo) Set(_ context.Context, key, val string) error {
	if m.settings == nil {
		m.settings = make(map[string]string)
	}
	m.settings[key] = val
	return nil
}

func (m *mockSettingRepo) GetAll(_ context.Context) ([]entities.PlatformSetting, error) {
	return nil, nil
}
