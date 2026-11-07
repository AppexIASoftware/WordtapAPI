package repositories

import (
	"context"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"gorm.io/gorm"
)

// loadAssignedCourses resolves all course IDs in one query, retaining the
// ordering configured by the teacher and omitting missing or duplicate IDs.
func (r *MySQLClassroomRepository) loadAssignedCourses(ctx context.Context, classrooms []entities.Classroom) error {
	ids := make(map[string]struct{})
	for _, classroom := range classrooms {
		for _, id := range assignedCourseIDs(classroom) {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	var courses []entities.Course
	if err := r.db.WithContext(ctx).
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", entities.ContentStatusPublished).Order("sort_order ASC, created_at ASC")
		}).
		Where("id IN ?", keys).Find(&courses).Error; err != nil {
		return err
	}
	byID := make(map[string]entities.Course, len(courses))
	for _, course := range courses {
		byID[course.ID] = course
	}
	for i := range classrooms {
		for _, id := range assignedCourseIDs(classrooms[i]) {
			if course, ok := byID[id]; ok {
				classrooms[i].ResolvedCourses = append(classrooms[i].ResolvedCourses, course)
			}
		}
	}
	return nil
}

func assignedCourseIDs(classroom entities.Classroom) []string {
	var raw []string
	if classroom.CourseID != nil {
		raw = append(raw, *classroom.CourseID)
	}
	if classroom.AssignedCourses != nil {
		raw = append(raw, strings.Split(*classroom.AssignedCourses, ",")...)
	}
	seen := make(map[string]bool)
	ids := make([]string, 0, len(raw))
	for _, value := range raw {
		id := strings.TrimSpace(value)
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}
