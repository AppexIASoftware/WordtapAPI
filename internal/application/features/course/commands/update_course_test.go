package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type updateMockCourseRepo struct {
	course *entities.Course
}

func (m *updateMockCourseRepo) FindByID(_ context.Context, id string) (*entities.Course, error) {
	return m.course, nil
}
func (m *updateMockCourseRepo) Create(_ context.Context, _ *entities.Course) error { return nil }
func (m *updateMockCourseRepo) FindByAuthor(_ context.Context, _ string) ([]entities.Course, error) {
	return nil, nil
}
func (m *updateMockCourseRepo) FindPublished(_ context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (m *updateMockCourseRepo) FindAll(_ context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (m *updateMockCourseRepo) Update(_ context.Context, c *entities.Course) error {
	m.course = c
	return nil
}
func (m *updateMockCourseRepo) Delete(_ context.Context, _ string) error { return nil }

func TestUpdateCourseMinPriceValidation(t *testing.T) {
	author := "teacher-1"

	t.Run("fails when updating price below minimum 2000 cents for teacher course", func(t *testing.T) {
		initialCourse := &entities.Course{
			ID:         "c-1",
			Title:      "Original",
			CreatedBy:  &author,
			OwnerType:  entities.OwnerTypeTeacher,
			AccessTier: entities.AccessTierCourse,
			PriceCents: 2000,
			Status:     entities.ContentStatusDraft,
		}
		repo := &updateMockCourseRepo{course: initialCourse}
		handler := commands.NewUpdateCourseHandler(repo)

		lowPrice := 1500
		_, err := handler.Handle(context.Background(), commands.UpdateCourseCommand{
			CourseID:      initialCourse.ID,
			PriceCents:    &lowPrice,
			RequesterID:   author,
			RequesterRole: entities.RoleInstructor,
		})
		if err == nil {
			t.Fatal("expected error when setting price below S/ 20.00, got nil")
		}
	})

	t.Run("succeeds when updating price to 2000 cents or higher", func(t *testing.T) {
		initialCourse := &entities.Course{
			ID:         "c-2",
			Title:      "Original",
			CreatedBy:  &author,
			OwnerType:  entities.OwnerTypeTeacher,
			AccessTier: entities.AccessTierCourse,
			PriceCents: 2000,
			Status:     entities.ContentStatusDraft,
		}
		repo := &updateMockCourseRepo{course: initialCourse}
		handler := commands.NewUpdateCourseHandler(repo)

		validPrice := 2500
		updated, err := handler.Handle(context.Background(), commands.UpdateCourseCommand{
			CourseID:      initialCourse.ID,
			PriceCents:    &validPrice,
			RequesterID:   author,
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.PriceCents != 2500 {
			t.Errorf("expected price 2500, got %d", updated.PriceCents)
		}
	})

	t.Run("enforces dynamic setting configured in platform_setting", func(t *testing.T) {
		initialCourse := &entities.Course{
			ID:         "c-3",
			Title:      "Original",
			CreatedBy:  &author,
			OwnerType:  entities.OwnerTypeTeacher,
			AccessTier: entities.AccessTierCourse,
			PriceCents: 3000,
			Status:     entities.ContentStatusDraft,
		}
		repo := &updateMockCourseRepo{course: initialCourse}
		settings := &mockSettingRepo{
			settings: map[string]string{
				commands.CourseMinPriceSettingKey: "3000",
			},
		}
		handler := commands.NewUpdateCourseHandler(repo).WithSettingRepo(settings)

		// 2500 is below configured 3000
		price2500 := 2500
		_, err := handler.Handle(context.Background(), commands.UpdateCourseCommand{
			CourseID:      initialCourse.ID,
			PriceCents:    &price2500,
			RequesterID:   author,
			RequesterRole: entities.RoleInstructor,
		})
		if err == nil {
			t.Fatal("expected error when setting price below configured 3000 cents, got nil")
		}

		// 3000 matches configured minimum
		price3000 := 3000
		updated, err := handler.Handle(context.Background(), commands.UpdateCourseCommand{
			CourseID:      initialCourse.ID,
			PriceCents:    &price3000,
			RequesterID:   author,
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error for 3000 cents: %v", err)
		}
		if updated.PriceCents != 3000 {
			t.Errorf("expected price 3000, got %d", updated.PriceCents)
		}
	})

	t.Run("free course does not require minimum price", func(t *testing.T) {
		initialCourse := &entities.Course{
			ID:         "c-4",
			Title:      "Free Course",
			CreatedBy:  &author,
			OwnerType:  entities.OwnerTypeTeacher,
			AccessTier: entities.AccessTierFree,
			PriceCents: 0,
			Status:     entities.ContentStatusDraft,
		}
		repo := &updateMockCourseRepo{course: initialCourse}
		handler := commands.NewUpdateCourseHandler(repo)

		newTitle := "Updated Free Course"
		updated, err := handler.Handle(context.Background(), commands.UpdateCourseCommand{
			CourseID:      initialCourse.ID,
			Title:         &newTitle,
			RequesterID:   author,
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.PriceCents != 0 {
			t.Errorf("expected price 0, got %d", updated.PriceCents)
		}
	})
}
