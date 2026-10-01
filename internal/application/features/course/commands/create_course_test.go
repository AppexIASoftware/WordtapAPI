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

func (m *mockCourseRepo) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepo) Update(ctx context.Context, course *entities.Course) error {
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
		if course.Slug == "" {
			t.Error("expected non-empty slug")
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
}
