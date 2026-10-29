package gamification

import (
	"encoding/json"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func formatTeacherGameDTO(g *entities.TeacherGame) map[string]any {
	courseTitle := ""
	if g.Course != nil {
		courseTitle = g.Course.Title
	}

	teacherName := ""
	teacherEmail := ""
	if g.Teacher != nil {
		teacherName = g.Teacher.Name
		teacherEmail = g.Teacher.Email
	}

	categoryName := ""
	if g.Category != nil {
		categoryName = g.Category.Name
	}

	accessTier := g.AccessTier
	if accessTier == "" {
		accessTier = entities.AccessTierFree
	}

	assignedCourseIDs := make([]string, 0)
	if g.AssignedCourseIDs != nil && *g.AssignedCourseIDs != "" {
		raw := strings.TrimSpace(*g.AssignedCourseIDs)
		if err := json.Unmarshal([]byte(raw), &assignedCourseIDs); err == nil {
			// JSON unmarshal succeeded
		} else {
			for _, part := range strings.Split(raw, ",") {
				if clean := strings.TrimSpace(part); clean != "" {
					assignedCourseIDs = append(assignedCourseIDs, clean)
				}
			}
		}
	}

	return map[string]any{
		"id":                      g.ID,
		"teacher_id":              g.TeacherID,
		"teacher_name":            teacherName,
		"teacher_email":           teacherEmail,
		"course_id":               g.CourseID,
		"course_title":            courseTitle,
		"category_id":             g.CategoryID,
		"category_name":           categoryName,
		"base_archetype":          g.BaseArchetype,
		"title":                   g.Title,
		"description":             g.Description,
		"level":                   g.Level,
		"access_tier":             accessTier,
		"time_limit_seconds":      g.TimeLimitSeconds,
		"allowed_mistakes":        g.AllowedMistakes,
		"reward_xp":               g.RewardXP,
		"item_count":              g.ItemCount,
		"source_type":             g.SourceType,
		"source_id":               g.SourceID,
		"custom_words_json":       g.CustomWordsJSON,
		"owner_type":              g.OwnerType,
		"is_global":               g.IsGlobal,
		"is_published_standalone": g.IsPublishedStandalone,
		"audience_scope":          g.AudienceScope,
		"assigned_course_ids":     assignedCourseIDs,
		"usage_count":             g.UsageCount,
		"is_active":               g.IsActive,
		"created_at":              g.CreatedAt.Format("02/01/2006"),
	}
}
