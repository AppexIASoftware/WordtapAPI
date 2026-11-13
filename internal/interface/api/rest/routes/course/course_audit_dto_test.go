package course

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestCourseDetailDTO_PublicVsAudit(t *testing.T) {
	c := &entities.Course{
		ID:     "c1",
		Title:  "curso prueba",
		Status: entities.ContentStatusInReview,
		Lessons: []entities.Lesson{
			{
				ID:       "l1",
				CourseID: "c1",
				Title:    "leccion 1",
				Status:   entities.ContentStatusDraft,
				Items: []entities.LessonItem{
					{ID: "i1", LessonID: "l1", ItemType: "word"},
				},
			},
			{
				ID:       "l2",
				CourseID: "c1",
				Title:    "leccion 2",
				Status:   entities.ContentStatusPublished,
				Items: []entities.LessonItem{
					{ID: "i2", LessonID: "l2", ItemType: "word"},
				},
			},
		},
	}

	pub := toPublicCourseDTO(c, true)
	if len(pub.Lessons) != 1 || pub.Lessons[0].ID != "l2" || len(pub.Lessons[0].Items) != 0 {
		t.Fatalf("expected only published lesson without items in public DTO, got %+v", pub.Lessons)
	}

	audit := toAuditCourseDTO(c)
	if len(audit.Lessons) != 2 {
		t.Fatalf("expected 2 lessons in audit DTO, got %d", len(audit.Lessons))
	}
	if len(audit.Lessons[0].Items) != 1 || audit.Lessons[0].Items[0].ID != "i1" {
		t.Fatalf("expected draft lesson items in audit DTO, got %+v", audit.Lessons[0].Items)
	}
}
