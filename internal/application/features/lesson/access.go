// Package lesson holds access adapters for lesson features.
package lesson

import (
	shared "github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// CanManageCourse checks if requester can view course lesson games.
func CanManageCourse(course *entities.Course, requesterID string, role entities.UserRole) bool {
	return shared.CanViewCourse(course, requesterID, role)
}

// CanLinkGameToCourse checks if requester can link games to course.
func CanLinkGameToCourse(course *entities.Course, requesterID string, role entities.UserRole) bool {
	return shared.CanLinkGameToCourse(course, requesterID, role)
}
