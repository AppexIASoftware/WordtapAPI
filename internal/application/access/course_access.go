// Package access defines course and lesson authorization rules.
package access

import (
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// EffectiveOwnerType returns course owner type, defaulting to teacher.
func EffectiveOwnerType(course *entities.Course) entities.OwnerType {
	if course == nil {
		return entities.OwnerTypeTeacher
	}
	if course.OwnerType.IsValid() {
		return course.OwnerType
	}
	return entities.OwnerTypeTeacher
}

// CanViewCourse checks read permissions for a course.
func CanViewCourse(course *entities.Course, requesterID string, role entities.UserRole) bool {
	if course == nil {
		return false
	}
	switch role {
	case entities.RoleAdmin, entities.RoleModerator:
		return true
	case entities.RoleInstructor:
		return EffectiveOwnerType(course) == entities.OwnerTypeTeacher &&
			course.CreatedBy != nil && *course.CreatedBy == requesterID
	default:
		return false
	}
}

// CanEditCourseContent checks write permissions for a course.
func CanEditCourseContent(course *entities.Course, requesterID string, role entities.UserRole) bool {
	if course == nil {
		return false
	}
	switch role {
	case entities.RoleAdmin:
		return EffectiveOwnerType(course) == entities.OwnerTypePlatform
	case entities.RoleInstructor:
		return EffectiveOwnerType(course) == entities.OwnerTypeTeacher &&
			course.CreatedBy != nil && *course.CreatedBy == requesterID
	default:
		return false
	}
}

// CanLinkGameToCourse checks permission to link a game to a course.
func CanLinkGameToCourse(course *entities.Course, requesterID string, role entities.UserRole) bool {
	if course == nil {
		return false
	}
	switch role {
	case entities.RoleAdmin:
		return EffectiveOwnerType(course) == entities.OwnerTypePlatform
	case entities.RoleInstructor:
		return EffectiveOwnerType(course) == entities.OwnerTypeTeacher &&
			course.CreatedBy != nil && *course.CreatedBy == requesterID
	default:
		return false
	}
}
