package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockCourseRepoForDetail struct {
	courses []entities.Course
}

func (m *mockCourseRepoForDetail) Create(ctx context.Context, course *entities.Course) error {
	m.courses = append(m.courses, *course)
	return nil
}
func (m *mockCourseRepoForDetail) FindByAuthor(ctx context.Context, authorID string) ([]entities.Course, error) {
	return nil, nil
}
func (m *mockCourseRepoForDetail) FindPublished(ctx context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (m *mockCourseRepoForDetail) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	for _, c := range m.courses {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockCourseRepoForDetail) Update(ctx context.Context, course *entities.Course) error {
	return nil
}
func (m *mockCourseRepoForDetail) Delete(ctx context.Context, id string) error {
	return nil
}

func TestGetCourseDetailHandler(t *testing.T) {
	authorID := "teacher-author-1"
	otherID := "teacher-other-2"

	repo := &mockCourseRepoForDetail{
		courses: []entities.Course{
			{
				ID:        "course-published",
				Title:     "Published Course",
				Status:    entities.ContentStatusPublished,
				CreatedBy: &authorID,
			},
			{
				ID:        "course-draft",
				Title:     "Draft Course",
				Status:    entities.ContentStatusDraft,
				CreatedBy: &authorID,
			},
			{
				ID:        "course-in-review",
				Title:     "In Review Course",
				Status:    entities.ContentStatusInReview,
				CreatedBy: &authorID,
			},
		},
	}

	handler := queries.NewGetCourseDetailHandler(repo)
	ctx := context.Background()

	t.Run("anonymous user can read published course", func(t *testing.T) {
		course, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID: "course-published",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if course.ID != "course-published" {
			t.Errorf("expected course-published, got %s", course.ID)
		}
	})

	t.Run("anonymous user cannot read draft course", func(t *testing.T) {
		_, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID: "course-draft",
		})
		if !errors.Is(err, queries.ErrCourseAccessDenied) {
			t.Fatalf("expected ErrCourseAccessDenied, got %v", err)
		}
	})

	t.Run("foreign teacher cannot read draft course", func(t *testing.T) {
		_, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID:      "course-draft",
			RequesterID:   otherID,
			RequesterRole: entities.RoleInstructor,
		})
		if !errors.Is(err, queries.ErrCourseAccessDenied) {
			t.Fatalf("expected ErrCourseAccessDenied, got %v", err)
		}
	})

	t.Run("author can read own draft course", func(t *testing.T) {
		course, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID:      "course-draft",
			RequesterID:   authorID,
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error for author: %v", err)
		}
		if course.ID != "course-draft" {
			t.Errorf("expected course-draft, got %s", course.ID)
		}
	})

	t.Run("admin can read any draft or in-review course", func(t *testing.T) {
		course, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID:      "course-in-review",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error for admin: %v", err)
		}
		if course.ID != "course-in-review" {
			t.Errorf("expected course-in-review, got %s", course.ID)
		}
	})

	t.Run("moderator can read course in review but not draft", func(t *testing.T) {
		inReviewCourse, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID:      "course-in-review",
			RequesterID:   "mod-1",
			RequesterRole: entities.RoleModerator,
		})
		if err != nil {
			t.Fatalf("unexpected error for moderator: %v", err)
		}
		if inReviewCourse.ID != "course-in-review" {
			t.Errorf("expected course-in-review, got %s", inReviewCourse.ID)
		}

		_, err = handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID:      "course-draft",
			RequesterID:   "mod-1",
			RequesterRole: entities.RoleModerator,
		})
		if !errors.Is(err, queries.ErrCourseAccessDenied) {
			t.Fatalf("expected ErrCourseAccessDenied for draft, got %v", err)
		}
	})

	t.Run("non existent course returns ErrCourseNotFound", func(t *testing.T) {
		_, err := handler.Handle(ctx, queries.GetCourseDetailQuery{
			CourseID: "does-not-exist",
		})
		if !errors.Is(err, queries.ErrCourseNotFound) {
			t.Fatalf("expected ErrCourseNotFound, got %v", err)
		}
	})
}
