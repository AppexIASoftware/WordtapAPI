package access

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// RBACSettingKey matches the key persisted by WordtapStudio in platform_settings.
const RBACSettingKey = "rbac_matrix_policies"

// RBACPermission identifies a granular policy key from the RBAC matrix.
type RBACPermission string

const (
	PermVaultAuthorItems                 RBACPermission = "vault:author_items"
	PermCoursesCreateDrafts              RBACPermission = "courses:create_drafts"
	PermCoursesAuthorFreeCourses         RBACPermission = "courses:author_free_courses"
	PermGamesTuneLevels                  RBACPermission = "games:tune_levels"
	PermModerationReviewCourses          RBACPermission = "moderation:review_courses"
	PermModerationManageTeacherApps      RBACPermission = "moderation:manage_teacher_apps"
	PermCohortsCreateClassrooms          RBACPermission = "cohorts:create_classrooms"
	PermNotificationsClassroomBroadcast  RBACPermission = "notifications:classroom_broadcast"
	PermNotificationsGlobalAnnouncements RBACPermission = "notifications:global_announcements"
	PermMonetizationCustomCoursePrice    RBACPermission = "monetization:custom_course_price"
	PermStudentsViewDirectory            RBACPermission = "students:view_directory"
	PermStudentsManageStatus             RBACPermission = "students:manage_status"
)

type rbacCell struct {
	val    bool
	locked bool
}

type rbacRow struct {
	student    rbacCell
	instructor rbacCell
	moderator  rbacCell
}

var defaultRBACMatrix = map[RBACPermission]rbacRow{
	PermVaultAuthorItems: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: true, locked: true},
		moderator:  rbacCell{val: true, locked: false},
	},
	PermCoursesCreateDrafts: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: true, locked: true},
		moderator:  rbacCell{val: true, locked: false},
	},
	PermCoursesAuthorFreeCourses: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: false, locked: false},
		moderator:  rbacCell{val: false, locked: false},
	},
	PermGamesTuneLevels: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: true, locked: false},
		moderator:  rbacCell{val: true, locked: false},
	},
	PermModerationReviewCourses: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: false, locked: true},
		moderator:  rbacCell{val: true, locked: false},
	},
	PermModerationManageTeacherApps: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: false, locked: true},
		moderator:  rbacCell{val: true, locked: false},
	},
	PermCohortsCreateClassrooms: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: true, locked: true},
		moderator:  rbacCell{val: false, locked: false},
	},
	PermNotificationsClassroomBroadcast: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: true, locked: false},
		moderator:  rbacCell{val: false, locked: false},
	},
	PermNotificationsGlobalAnnouncements: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: false, locked: false},
		moderator:  rbacCell{val: true, locked: false},
	},
	PermMonetizationCustomCoursePrice: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: true, locked: false},
		moderator:  rbacCell{val: false, locked: true},
	},
	PermStudentsViewDirectory: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: false, locked: true},
		moderator:  rbacCell{val: false, locked: false},
	},
	PermStudentsManageStatus: {
		student:    rbacCell{val: false, locked: true},
		instructor: rbacCell{val: false, locked: true},
		moderator:  rbacCell{val: false, locked: false},
	},
}

// RBACPolicyEvaluator resolves dynamic role permissions backed by platform_settings.
type RBACPolicyEvaluator struct {
	settings repositories.PlatformSettingRepository
}

// NewRBACPolicyEvaluator constructs an evaluator using the given settings repository.
func NewRBACPolicyEvaluator(settings repositories.PlatformSettingRepository) *RBACPolicyEvaluator {
	return &RBACPolicyEvaluator{settings: settings}
}

// IsAllowed evaluates whether role holds perm, respecting locked structural invariants
// and falling back to safe defaults if settings are absent or malformed.
func (e *RBACPolicyEvaluator) IsAllowed(ctx context.Context, role entities.UserRole, perm RBACPermission) bool {
	if role == entities.RoleAdmin {
		return true
	}

	row, exists := defaultRBACMatrix[perm]
	if !exists {
		return false
	}

	var cell rbacCell
	var roleKey string
	switch role {
	case entities.RoleStudent:
		cell = row.student
		roleKey = "student"
	case entities.RoleInstructor:
		cell = row.instructor
		roleKey = "instructor"
	case entities.RoleModerator:
		cell = row.moderator
		roleKey = "moderator"
	default:
		return false
	}

	if cell.locked || e == nil || e.settings == nil {
		return cell.val
	}

	setting, err := e.settings.Get(ctx, RBACSettingKey)
	if err != nil || setting == nil || strings.TrimSpace(setting.Value) == "" {
		return cell.val
	}

	var overrides map[string]map[string]bool
	if err := json.Unmarshal([]byte(setting.Value), &overrides); err != nil {
		return cell.val
	}

	if permOverrides, ok := overrides[string(perm)]; ok {
		if val, hasRole := permOverrides[roleKey]; hasRole {
			return val
		}
	}

	return cell.val
}

// PermissionsForRole returns a map of all RBAC permission keys evaluated for the given role.
func (e *RBACPolicyEvaluator) PermissionsForRole(ctx context.Context, role entities.UserRole) map[string]bool {
	result := make(map[string]bool, len(defaultRBACMatrix))
	for perm := range defaultRBACMatrix {
		result[string(perm)] = e.IsAllowed(ctx, role, perm)
	}
	return result
}
