package commands

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func availableLessonIDs(ctx context.Context, repo CourseLookup, courses []string) (map[string]bool, error) {
	available := make(map[string]bool)
	for _, id := range courses {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if repo == nil {
			return nil, apperrors.New("No se pudieron validar las lecciones.")
		}
		course, err := repo.FindByID(ctx, id)
		if err != nil || course == nil {
			return nil, apperrors.New("No se pudo validar el curso seleccionado.")
		}
		for _, lesson := range course.Lessons {
			if lesson.CourseID == course.ID && lesson.Status == entities.ContentStatusPublished {
				available[lesson.ID] = true
			}
		}
	}
	return available, nil
}

func encodeLessonSelection(ctx context.Context, repo CourseLookup, courses, lessons []string) (*string, error) {
	if lessons == nil {
		return nil, nil
	}
	available, err := availableLessonIDs(ctx, repo, courses)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(lessons))
	for _, id := range lessons {
		if id == "" || id != strings.TrimSpace(id) || !available[id] || seen[id] {
			return nil, apperrors.New("Las lecciones deben estar publicadas y pertenecer a los cursos seleccionados, sin duplicados.")
		}
		seen[id] = true
	}
	data, err := json.Marshal(lessons)
	if err != nil {
		return nil, err
	}
	value := string(data)
	return &value, nil
}

func retainAssignedLessons(ctx context.Context, repo CourseLookup, courses []string, stored *string) (*string, error) {
	var ids []string
	if err := json.Unmarshal([]byte(*stored), &ids); err != nil {
		return nil, err
	}
	available, err := availableLessonIDs(ctx, repo, courses)
	if err != nil {
		return nil, err
	}
	kept := make([]string, 0, len(ids))
	for _, id := range ids {
		if available[id] {
			kept = append(kept, id)
		}
	}
	data, err := json.Marshal(kept)
	if err != nil {
		return nil, err
	}
	value := string(data)
	return &value, nil
}
