package commands_test

import (
	"context"
	"testing"

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/lessontest"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type stubBankRepoForLesson struct {
	items map[string]*entities.ContentBankItem
}

func (s *stubBankRepoForLesson) ListTypes(context.Context) ([]entities.ContentBankType, error) {
	return nil, nil
}
func (s *stubBankRepoForLesson) FindTypeBySlug(context.Context, string) (*entities.ContentBankType, error) {
	return nil, nil
}
func (s *stubBankRepoForLesson) ListCategories(context.Context) ([]entities.ContentCategory, error) {
	return nil, nil
}
func (s *stubBankRepoForLesson) CreateCategory(context.Context, *entities.ContentCategory) error {
	return nil
}
func (s *stubBankRepoForLesson) ListItems(context.Context, repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	return nil, 0, nil
}
func (s *stubBankRepoForLesson) FindItemByID(_ context.Context, id string) (*entities.ContentBankItem, error) {
	return s.items[id], nil
}
func (s *stubBankRepoForLesson) CreateItem(context.Context, *entities.ContentBankItem) error {
	return nil
}
func (s *stubBankRepoForLesson) UpdateItem(context.Context, *entities.ContentBankItem) error {
	return nil
}
func (s *stubBankRepoForLesson) DeleteItem(context.Context, string) error { return nil }

func TestSaveLessonItemHandler_PrivateVaultGuard(t *testing.T) {
	fixture := lessontest.NewFixture()
	ownerOther := "teacher-2"
	ownerSelf := "teacher-1"
	bankRepo := &stubBankRepoForLesson{
		items: map[string]*entities.ContentBankItem{
			"vault-other": {ID: "vault-other", CreatedBy: &ownerOther},
			"vault-own":   {ID: "vault-own", CreatedBy: &ownerSelf},
		},
	}

	h := lessonCmd.NewSaveLessonItemHandler(fixture.Lessons, fixture.Courses).WithContentBankRepo(bankRepo)

	t.Run("rejects linking another teacher private vault item", func(t *testing.T) {
		foreignID := "vault-other"
		_, err := h.Handle(context.Background(), lessonCmd.SaveLessonItemCommand{
			LessonID:          "lesson-teacher",
			ContentBankItemID: &foreignID,
			ItemType:          entities.LessonContentTypeWord,
			RequesterID:       "teacher-1",
			RequesterRole:     entities.RoleInstructor,
		})
		if err == nil {
			t.Fatal("expected error linking foreign private vault item")
		}
	})

	t.Run("rejects nonexistent vault item", func(t *testing.T) {
		missingID := "vault-missing"
		_, err := h.Handle(context.Background(), lessonCmd.SaveLessonItemCommand{
			LessonID:          "lesson-teacher",
			ContentBankItemID: &missingID,
			ItemType:          entities.LessonContentTypeWord,
			RequesterID:       "teacher-1",
			RequesterRole:     entities.RoleInstructor,
		})
		if err == nil {
			t.Fatal("expected error linking nonexistent vault item")
		}
	})

	t.Run("allows linking own private vault item", func(t *testing.T) {
		ownID := "vault-own"
		item, err := h.Handle(context.Background(), lessonCmd.SaveLessonItemCommand{
			LessonID:          "lesson-teacher",
			ContentBankItemID: &ownID,
			ItemType:          entities.LessonContentTypeWord,
			RequesterID:       "teacher-1",
			RequesterRole:     entities.RoleInstructor,
		})
		if err != nil || item == nil {
			t.Fatalf("expected own private vault item to link cleanly, got err=%v", err)
		}
	})
}
