package queries

import (
	"context"
	"time"
)

// DashboardUserStats holds real user counts from MySQL.
type DashboardUserStats struct {
	TotalUsers       int64 `json:"total_users"`
	Students         int64 `json:"students"`
	Instructors      int64 `json:"instructors"`
	Moderators       int64 `json:"moderators"`
	Admins           int64 `json:"admins"`
	FreeStudents     int64 `json:"free_students"`
	CourseStudents   int64 `json:"course_students"`
	SubStudents      int64 `json:"subscription_students"`
}

// DashboardCatalogStats holds real catalog, moderation, and content counts.
type DashboardCatalogStats struct {
	TotalCourses        int64 `json:"total_courses"`
	PublishedCourses    int64 `json:"published_courses"`
	InReviewCourses     int64 `json:"in_review_courses"`
	DraftCourses        int64 `json:"draft_courses"`
	PendingTeacherApps  int64 `json:"pending_teacher_apps"`
	ApprovedTeacherApps int64 `json:"approved_teacher_apps"`
	TotalVaultItems     int64 `json:"total_vault_items"`
	TotalGames          int64 `json:"total_games"`
	TotalClassrooms     int64 `json:"total_classrooms"`
}

// DashboardEventItem represents a real recent system event from MySQL.
type DashboardEventItem struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Summary   string    `json:"summary"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminDashboardResponse is returned by GET /api/v1/admin/dashboard.
type AdminDashboardResponse struct {
	Users        DashboardUserStats    `json:"users"`
	Catalog      DashboardCatalogStats `json:"catalog"`
	RecentEvents []DashboardEventItem  `json:"recent_events"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

// DashboardReader abstracts read-only aggregation queries for the admin dashboard.
type DashboardReader interface {
	GetAdminDashboard(ctx context.Context) (*AdminDashboardResponse, error)
}

// GetAdminDashboardHandler executes the admin dashboard aggregation query.
type GetAdminDashboardHandler struct {
	reader DashboardReader
}

// NewGetAdminDashboardHandler creates a new GetAdminDashboardHandler.
func NewGetAdminDashboardHandler(reader DashboardReader) *GetAdminDashboardHandler {
	return &GetAdminDashboardHandler{reader: reader}
}

// Handle fetches aggregated dashboard statistics and recent events.
func (h *GetAdminDashboardHandler) Handle(ctx context.Context) (*AdminDashboardResponse, error) {
	return h.reader.GetAdminDashboard(ctx)
}
