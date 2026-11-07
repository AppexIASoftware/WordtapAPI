package course

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestPublicCourseDTOProtectsTeacherLessons(t *testing.T) {
	course := entities.Course{OwnerType: entities.OwnerTypeTeacher, Lessons: []entities.Lesson{
		{ID: "published", Status: entities.ContentStatusPublished},
		{ID: "draft", Status: entities.ContentStatusDraft},
	}}
	public := ToPublicCourseDTO(&course)
	if len(public.Lessons) != 0 || public.LessonsCount != 1 {
		t.Fatalf("unexpected public projection: %+v", public)
	}
	owned := toPublicCourseDTO(&course, true)
	if len(owned.Lessons) != 1 || owned.Lessons[0].ID != "published" {
		t.Fatalf("unexpected owned projection: %+v", owned)
	}
}
