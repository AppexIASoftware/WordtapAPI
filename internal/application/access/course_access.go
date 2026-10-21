// Package access holds the ownership rules shared by course and lesson
// features. Keeping them in one place prevents the platform/teacher boundary
// from drifting between endpoints.
package access

import (
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// EffectiveOwnerType returns the ownership stored on the course, falling back
// to teacher ownership for legacy rows that predate the owner_type column.
func EffectiveOwnerType(course *entities.Course) entities.OwnerType {
	if course == nil {
		return entities.OwnerTypeTeacher
	}
	if course.OwnerType.IsValid() {
		return course.OwnerType
	}
	return entities.OwnerTypeTeacher
}

// CanViewCourse reports whether the requester may inspect a course and its
// content. Administrators and moderators supervise any course; an instructor
// may only inspect their own teacher-owned course.
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

// CanEditCourseContent reports whether the requester may change a course,
// its lessons, cards, or pricing. Administrators own platform courses and
// instructors own their teacher courses. Moderators supervise but do not edit.
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

// CanLinkGameToCourse applies the stricter linking rule: an administrator only
// links games to platform-owned courses, while an instructor only links games
// to their own teacher-owned courses.
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
