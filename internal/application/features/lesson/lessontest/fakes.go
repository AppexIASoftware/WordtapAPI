// Package lessontest provides in-memory test doubles.
package lessontest

import (
	"context"
	"strconv"

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func StrPtr(value string) *string { return &value }
func IntPtr(value int) *int       { return &value }
func BoolPtr(value bool) *bool    { return &value }

type FakeLessonRepo struct {
	Lessons map[string]*entities.Lesson
}

func (f *FakeLessonRepo) FindByIDOrSlug(context.Context, string) (*entities.Lesson, error) {
	return nil, nil
}
func (f *FakeLessonRepo) FindByCourseID(context.Context, string) ([]entities.Lesson, error) {
	return nil, nil
}
func (f *FakeLessonRepo) FindByID(_ context.Context, id string) (*entities.Lesson, error) {
	return f.Lessons[id], nil
}
func (f *FakeLessonRepo) Create(context.Context, *entities.Lesson) error { return nil }
func (f *FakeLessonRepo) Update(context.Context, *entities.Lesson) error { return nil }
func (f *FakeLessonRepo) Delete(context.Context, string) error           { return nil }
func (f *FakeLessonRepo) SaveItem(context.Context, *entities.LessonItem) error {
	return nil
}
func (f *FakeLessonRepo) DeleteItem(context.Context, string) error { return nil }

type FakeCourseRepo struct {
	Courses map[string]*entities.Course
}

func (f *FakeCourseRepo) Create(context.Context, *entities.Course) error { return nil }
func (f *FakeCourseRepo) FindByAuthor(context.Context, string) ([]entities.Course, error) {
	return nil, nil
}
func (f *FakeCourseRepo) FindPublished(context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (f *FakeCourseRepo) FindAll(context.Context) ([]entities.Course, error) {
	return nil, nil
}
func (f *FakeCourseRepo) FindByID(_ context.Context, id string) (*entities.Course, error) {
	return f.Courses[id], nil
}
func (f *FakeCourseRepo) Update(context.Context, *entities.Course) error { return nil }
func (f *FakeCourseRepo) Delete(context.Context, string) error           { return nil }

type FakeGameRepo struct {
	Games map[string]*entities.TeacherGame
	Usage map[string]int
}

func (f *FakeGameRepo) Create(context.Context, *entities.TeacherGame) error { return nil }
func (f *FakeGameRepo) FindByID(_ context.Context, id string) (*entities.TeacherGame, error) {
	return f.Games[id], nil
}
func (f *FakeGameRepo) FindByTeacher(context.Context, string) ([]entities.TeacherGame, error) {
	return nil, nil
}
func (f *FakeGameRepo) FindGlobal(context.Context) ([]entities.TeacherGame, error) {
	return nil, nil
}
func (f *FakeGameRepo) FindStandaloneByTeachers(context.Context, []string) ([]entities.TeacherGame, error) {
	return nil, nil
}
func (f *FakeGameRepo) FindAll(context.Context) ([]entities.TeacherGame, error) {
	return nil, nil
}
func (f *FakeGameRepo) Update(context.Context, *entities.TeacherGame) error { return nil }
func (f *FakeGameRepo) Delete(context.Context, string) error                { return nil }
func (f *FakeGameRepo) CountLessonUsage(_ context.Context, gameID string) (int, error) {
	return f.Usage[gameID], nil
}

type FakeLinkRepo struct {
	Links []entities.LessonGame
}

func (f *FakeLinkRepo) Create(_ context.Context, link *entities.LessonGame) error {
	if link.ID == "" {
		link.ID = "link-" + strconv.Itoa(len(f.Links)+1)
	}
	f.Links = append(f.Links, *link)
	return nil
}

func (f *FakeLinkRepo) FindByLesson(_ context.Context, lessonID string) ([]entities.LessonGame, error) {
	result := make([]entities.LessonGame, 0)
	for _, link := range f.Links {
		if link.LessonID == lessonID {
			result = append(result, link)
		}
	}
	return result, nil
}

func (f *FakeLinkRepo) FindByLessonAndGame(_ context.Context, lessonID, gameID string) (*entities.LessonGame, error) {
	for i := range f.Links {
		if f.Links[i].LessonID == lessonID && f.Links[i].TeacherGameID == gameID {
			link := f.Links[i]
			return &link, nil
		}
	}
	return nil, nil
}

func (f *FakeLinkRepo) Update(_ context.Context, link *entities.LessonGame) error {
	for i := range f.Links {
		if f.Links[i].ID == link.ID {
			f.Links[i] = *link
			return nil
		}
	}
	return nil
}

func (f *FakeLinkRepo) Delete(_ context.Context, lessonID, gameID string) error {
	for i := range f.Links {
		if f.Links[i].LessonID == lessonID && f.Links[i].TeacherGameID == gameID {
			f.Links = append(f.Links[:i], f.Links[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *FakeLinkRepo) NextSortOrder(_ context.Context, lessonID string) (int, error) {
	maxOrder := -1
	for _, link := range f.Links {
		if link.LessonID == lessonID && link.SortOrder > maxOrder {
			maxOrder = link.SortOrder
		}
	}
	return maxOrder + 1, nil
}

func (f *FakeLinkRepo) CountByGameIDs(_ context.Context, ids []string) (map[string]int, error) {
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}

	counts := make(map[string]int, len(ids))
	for _, link := range f.Links {
		if _, ok := wanted[link.TeacherGameID]; ok {
			counts[link.TeacherGameID]++
		}
	}
	return counts, nil
}

// Fixture assembles repositories and handlers for lesson game tests.
type Fixture struct {
	Lessons *FakeLessonRepo
	Courses *FakeCourseRepo
	Games   *FakeGameRepo
	Links   *FakeLinkRepo
}

func NewFixture() *Fixture {
	return &Fixture{
		Lessons: &FakeLessonRepo{Lessons: map[string]*entities.Lesson{
			"lesson-platform": {
				ID:       "lesson-platform",
				CourseID: "course-platform",
				Status:   entities.ContentStatusPublished,
			},
			"lesson-teacher": {
				ID:       "lesson-teacher",
				CourseID: "course-teacher",
				Status:   entities.ContentStatusPublished,
			},
			"lesson-draft": {
				ID:       "lesson-draft",
				CourseID: "course-draft",
				Status:   entities.ContentStatusDraft,
			},
		}},
		Courses: &FakeCourseRepo{Courses: map[string]*entities.Course{
			"course-platform": {
				ID:        "course-platform",
				OwnerType: entities.OwnerTypePlatform,
				Status:    entities.ContentStatusPublished,
				Level:     "B1",
				CreatedBy: StrPtr("admin-1"),
			},
			"course-teacher": {
				ID:        "course-teacher",
				OwnerType: entities.OwnerTypeTeacher,
				Status:    entities.ContentStatusPublished,
				Level:     "B1",
				CreatedBy: StrPtr("teacher-1"),
			},
			"course-draft": {
				ID:        "course-draft",
				OwnerType: entities.OwnerTypeTeacher,
				Status:    entities.ContentStatusDraft,
				Level:     "B1",
				CreatedBy: StrPtr("teacher-1"),
			},
		}},
		Games: &FakeGameRepo{Games: map[string]*entities.TeacherGame{
			"game-global": {
				ID:        "game-global",
				TeacherID: "admin-1",
				OwnerType: entities.OwnerTypePlatform,
				IsGlobal:  true,
				IsActive:  true,
				Level:     "B1",
				Title:     "Actividad Global",
			},
			"game-global-2": {
				ID:        "game-global-2",
				TeacherID: "admin-1",
				OwnerType: entities.OwnerTypePlatform,
				IsGlobal:  true,
				IsActive:  true,
				Level:     "B1",
				Title:     "Actividad Global 2",
			},
			"game-teacher": {
				ID:        "game-teacher",
				TeacherID: "teacher-1",
				OwnerType: entities.OwnerTypeTeacher,
				IsActive:  true,
				Level:     "B1",
				Title:     "Actividad Propia",
			},
			"game-other": {
				ID:        "game-other",
				TeacherID: "teacher-2",
				OwnerType: entities.OwnerTypeTeacher,
				IsActive:  true,
				Level:     "B2",
				Title:     "Actividad Ajena",
			},
			"game-advanced": {
				ID:        "game-advanced",
				TeacherID: "teacher-1",
				OwnerType: entities.OwnerTypeTeacher,
				IsActive:  true,
				Level:     "C1",
				Title:     "Actividad Avanzada",
			},
			"game-basic": {
				ID:        "game-basic",
				TeacherID: "teacher-1",
				OwnerType: entities.OwnerTypeTeacher,
				IsActive:  true,
				Level:     "A1",
				Title:     "Actividad de Repaso",
			},
		}},
		Links: &FakeLinkRepo{},
	}
}

func (f *Fixture) LinkHandler() *lessonCmd.LinkLessonGameHandler {
	return lessonCmd.NewLinkLessonGameHandler(f.Lessons, f.Courses, f.Games, f.Links)
}

func (f *Fixture) UpdateHandler() *lessonCmd.UpdateLessonGameHandler {
	return lessonCmd.NewUpdateLessonGameHandler(f.Lessons, f.Courses, f.Links)
}

func (f *Fixture) UnlinkHandler() *lessonCmd.UnlinkLessonGameHandler {
	return lessonCmd.NewUnlinkLessonGameHandler(f.Lessons, f.Courses, f.Links)
}

func (f *Fixture) ListHandler() *lessonQuery.ListLessonGamesHandler {
	return lessonQuery.NewListLessonGamesHandler(f.Lessons, f.Courses, f.Links)
}
