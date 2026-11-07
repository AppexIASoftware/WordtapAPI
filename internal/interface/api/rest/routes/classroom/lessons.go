package classroom

import (
	"encoding/json"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// A nil selection preserves the legacy behavior of including all course lessons.
func classroomLessonIDs(classroom *entities.Classroom) []string {
	if classroom.AssignedLessons == nil {
		return nil
	}
	ids := make([]string, 0)
	if err := json.Unmarshal([]byte(*classroom.AssignedLessons), &ids); err != nil {
		return []string{}
	}
	return ids
}

func studentCourseLessons(classroom *entities.Classroom, course entities.Course) []map[string]any {
	selected := classroomLessonIDs(classroom)
	allowed := make(map[string]bool, len(selected))
	for _, id := range selected {
		allowed[id] = true
	}
	lessons := make([]map[string]any, 0)
	for _, lesson := range course.Lessons {
		if lesson.Status != entities.ContentStatusPublished || (selected != nil && !allowed[lesson.ID]) {
			continue
		}
		lessons = append(lessons, map[string]any{"id": lesson.ID, "title": lesson.Title, "sortOrder": lesson.SortOrder})
	}
	return lessons
}
