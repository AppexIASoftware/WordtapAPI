package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type gameLessonRepo struct {
	repositories.LessonRepository
	lesson *entities.Lesson
}

func (r gameLessonRepo) FindByID(context.Context, string) (*entities.Lesson, error) {
	return r.lesson, nil
}

type gamePurchases struct{}

func (gamePurchases) HasPurchase(_ context.Context, student, course string) (bool, error) {
	return student == "buyer" && course == "course", nil
}

type gameLinks struct {
	links []entities.LessonGame
	err   error
}

func (r *gameLinks) FindLessonsByGame(context.Context, string) ([]entities.LessonGame, error) {
	return r.links, r.err
}

func TestCourseGameAccess(t *testing.T) {
	source := "lesson"
	course := &entities.Course{ID: "course", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished}
	lesson := &entities.Lesson{ID: source, CourseID: course.ID, Course: course, Status: entities.ContentStatusPublished}
	links := &gameLinks{}
	guard := NewCourseGameAccess(gameLessonRepo{lesson: lesson}, links, access.NewContentPolicy(gamePurchases{}, true))
	game := &entities.TeacherGame{ID: "game", SourceType: "course_lesson", SourceID: &source}
	if err := guard.authorize(context.Background(), game, "buyer"); err != nil {
		t.Fatal(err)
	}
	if err := guard.authorize(context.Background(), game, "other"); !errors.Is(err, access.ErrContentDenied) {
		t.Fatalf("expected denial: %v", err)
	}
	lesson.Status = entities.ContentStatusDraft
	if err := guard.authorize(context.Background(), game, "buyer"); !errors.Is(err, access.ErrContentDenied) {
		t.Fatalf("expected draft denial: %v", err)
	}
	game.SourceID = nil
	if err := guard.authorize(context.Background(), game, "other"); err != nil {
		t.Fatalf("standalone classroom game: %v", err)
	}
}

func TestCourseGameAccessLinkedLessons(t *testing.T) {
	paid := &entities.Course{ID: "course", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished}
	other := &entities.Course{ID: "other", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished}
	platform := &entities.Course{ID: "platform", OwnerType: entities.OwnerTypePlatform, Status: entities.ContentStatusPublished}
	link := func(id string, course *entities.Course, status entities.ContentStatus) entities.LessonGame {
		return entities.LessonGame{TeacherGameID: "game", LessonID: id, Lesson: &entities.Lesson{ID: id, CourseID: course.ID, Course: course, Status: status}}
	}
	broken := entities.LessonGame{TeacherGameID: "game", LessonID: "missing"}
	storageErr := errors.New("storage unavailable")
	for _, tc := range []struct {
		name       string
		links      []entities.LessonGame
		storageErr error
		student    string
		want       error
	}{
		{"independent without links", nil, nil, "other", nil},
		{"linked without purchase", []entities.LessonGame{link("l1", paid, entities.ContentStatusPublished)}, nil, "other", access.ErrContentDenied},
		{"linked with purchase", []entities.LessonGame{link("l1", paid, entities.ContentStatusPublished)}, nil, "buyer", nil},
		{"multiple courses require every entitlement", []entities.LessonGame{link("l1", paid, entities.ContentStatusPublished), link("l2", other, entities.ContentStatusPublished)}, nil, "buyer", access.ErrContentDenied},
		{"platform course", []entities.LessonGame{link("l3", platform, entities.ContentStatusPublished)}, nil, "other", nil},
		{"draft lesson", []entities.LessonGame{link("l1", paid, entities.ContentStatusDraft)}, nil, "buyer", access.ErrContentDenied},
		{"broken relation", []entities.LessonGame{broken}, nil, "buyer", access.ErrContentDenied},
		{"lookup failure", nil, storageErr, "buyer", storageErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			guard := NewCourseGameAccess(nil, &gameLinks{links: tc.links, err: tc.storageErr}, access.NewContentPolicy(gamePurchases{}, true))
			err := guard.authorize(context.Background(), &entities.TeacherGame{ID: "game", SourceType: "custom_list"}, tc.student)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	if err := (*CourseGameAccess)(nil).authorize(context.Background(), &entities.TeacherGame{ID: "game"}, "buyer"); !errors.Is(err, access.ErrContentDenied) {
		t.Fatalf("missing guard must deny: %v", err)
	}
}
