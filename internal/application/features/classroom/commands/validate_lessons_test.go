package commands

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestEncodeLessonSelection(t *testing.T) {
	repo := courseLookupStub{course: &entities.Course{ID: "course", Lessons: []entities.Lesson{
		{ID: "published", CourseID: "course", Status: entities.ContentStatusPublished},
		{ID: "draft", CourseID: "course", Status: entities.ContentStatusDraft},
		{ID: "foreign", CourseID: "other", Status: entities.ContentStatusPublished},
	}}}
	for _, tc := range []struct {
		name             string
		courses, lessons []string
		want             string
		invalid          bool
	}{
		{"legacy omitted", []string{"course"}, nil, "", false},
		{"explicit empty", []string{"course"}, []string{}, "[]", false},
		{"published", []string{"course"}, []string{"published"}, `["published"]`, false},
		{"draft", []string{"course"}, []string{"draft"}, "", true},
		{"foreign course", []string{"course"}, []string{"foreign"}, "", true},
		{"missing", []string{"course"}, []string{"missing"}, "", true},
		{"duplicate", []string{"course"}, []string{"published", "published"}, "", true},
		{"no course", nil, []string{"published"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeLessonSelection(context.Background(), repo, tc.courses, tc.lessons)
			if (err != nil) != tc.invalid {
				t.Fatalf("error = %v", err)
			}
			if tc.lessons == nil && got != nil {
				t.Fatalf("legacy selection = %v", *got)
			}
			if !tc.invalid && tc.lessons != nil && (got == nil || *got != tc.want) {
				t.Fatalf("selection = %v, want %s", got, tc.want)
			}
		})
	}
}

func TestRetainAssignedLessonsOnCourseChange(t *testing.T) {
	stored := `["kept","removed"]`
	repo := courseLookupStub{course: &entities.Course{ID: "course", Lessons: []entities.Lesson{{ID: "kept", CourseID: "course", Status: entities.ContentStatusPublished}}}}
	got, err := retainAssignedLessons(context.Background(), repo, []string{"course"}, &stored)
	if err != nil || got == nil || *got != `["kept"]` {
		t.Fatalf("selection = %v, error = %v", got, err)
	}
}
