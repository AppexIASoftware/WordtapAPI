package classroom

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestStudentCourseLessons(t *testing.T) {
	selected := `["two"]`
	empty := `[]`
	course := entities.Course{Lessons: []entities.Lesson{
		{ID: "one", Status: entities.ContentStatusPublished},
		{ID: "two", Status: entities.ContentStatusPublished},
		{ID: "draft", Status: entities.ContentStatusDraft},
	}}
	for _, tc := range []struct {
		name      string
		selection *string
		count     int
		first     string
	}{
		{"legacy", nil, 2, "one"}, {"explicit subset", &selected, 1, "two"}, {"explicit empty", &empty, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := studentCourseLessons(&entities.Classroom{AssignedLessons: tc.selection}, course)
			if len(got) != tc.count {
				t.Fatalf("got %d lessons, want %d", len(got), tc.count)
			}
			if tc.count > 0 && got[0]["id"] != tc.first {
				t.Fatalf("first = %v", got[0])
			}
		})
	}
}

func TestUnpurchasedCourseIncludesOnlySelectedPublishedLessonPreview(t *testing.T) {
	selection := `["selected"]`
	classroom := &entities.Classroom{AssignedLessons: &selection, ResolvedCourses: []entities.Course{{
		ID: "course", Status: entities.ContentStatusPublished,
		Lessons: []entities.Lesson{{ID: "selected", Title: "Selected", Status: entities.ContentStatusPublished}, {ID: "draft", Status: entities.ContentStatusDraft}},
	}}}
	dto := formatStudentClassroomDTO(classroom, "student", map[string]bool{"course": false})
	courses := dto["assignedCourses"].([]map[string]any)
	lessons, ok := courses[0]["lessons"].([]map[string]any)
	if !ok || len(lessons) != 1 || lessons[0]["title"] != "Selected" || len(lessons[0]) != 3 {
		t.Fatalf("unexpected preview: %#v", courses[0]["lessons"])
	}
}
