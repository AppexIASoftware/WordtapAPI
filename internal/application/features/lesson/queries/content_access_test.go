package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/lessontest"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type detailRepo struct {
	repositories.LessonRepository
	lesson *entities.Lesson
}

func (r detailRepo) FindByIDOrSlug(_ context.Context, id string) (*entities.Lesson, error) {
	if id == r.lesson.ID || id == r.lesson.Slug {
		return r.lesson, nil
	}
	return nil, errors.New("not found")
}

type ownedPurchase struct{}

func (ownedPurchase) HasPurchase(_ context.Context, user, course string) (bool, error) {
	return user == "buyer" && course == "course", nil
}

func TestLessonDetailChecksPurchaseForIDAndSlug(t *testing.T) {
	owner := "teacher"
	course := &entities.Course{ID: "course", OwnerType: entities.OwnerTypeTeacher, CreatedBy: &owner, Status: entities.ContentStatusPublished}
	lesson := &entities.Lesson{ID: "id", Slug: "slug", CourseID: course.ID, Course: course, Status: entities.ContentStatusPublished}
	h := lessonQuery.NewGetLessonDetailHandler(detailRepo{lesson: lesson}).WithContentPolicy(access.NewContentPolicy(ownedPurchase{}, true))
	for _, identifier := range []string{"id", "slug"} {
		for _, tc := range []struct {
			user    string
			allowed bool
		}{{"buyer", true}, {"other", false}} {
			t.Run(identifier+"/"+tc.user, func(t *testing.T) {
				dto, err := h.Handle(context.Background(), lessonQuery.GetLessonDetailQuery{Identifier: identifier, RequesterID: tc.user, RequesterRole: entities.RoleStudent})
				if tc.allowed && (err != nil || dto == nil) {
					t.Fatalf("expected access: %v", err)
				}
				if !tc.allowed && (!errors.Is(err, access.ErrContentDenied) || dto != nil) {
					t.Fatalf("expected denial, got %+v %v", dto, err)
				}
			})
		}
	}
}

func TestLessonGamesDenyUnpurchasedStudent(t *testing.T) {
	fixture := lessontest.NewFixture()
	_, err := fixture.ListHandler().Handle(context.Background(), lessonQuery.ListLessonGamesQuery{LessonID: "lesson-teacher", RequesterID: "other", RequesterRole: entities.RoleStudent})
	if !errors.Is(err, lessonQuery.ErrLessonGamesAccessDenied) {
		t.Fatalf("expected denial, got %v", err)
	}
}
