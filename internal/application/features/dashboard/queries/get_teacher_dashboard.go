package queries

import (
	"context"
	"time"
)

// TeacherDashboardResponse is returned by GET /api/v1/teacher/dashboard.
type TeacherDashboardResponse struct {
	Period              string    `json:"period"`
	ActiveStudents      int64     `json:"active_students"`
	AvgStreakDays       float64   `json:"avg_streak_days"`
	TotalStudents       int64     `json:"total_students"`
	RetentionPercent    float64   `json:"retention_percent"`
	VaultItems          int64     `json:"vault_items"`
	VaultBanks          int64     `json:"vault_banks"`
	RevenueCents        int64     `json:"revenue_cents"`
	PaidOrders          int64     `json:"paid_orders"`
	ActiveSubscriptions int64     `json:"active_subscriptions"`
	GeneratedAt         time.Time `json:"generated_at"`
}

// TeacherDashboardReader abstracts read-only metrics for the teacher dashboard.
type TeacherDashboardReader interface {
	GetTeacherDashboard(ctx context.Context, instructorID, period string) (*TeacherDashboardResponse, error)
}

// GetTeacherDashboardHandler executes the teacher dashboard metrics query.
type GetTeacherDashboardHandler struct {
	reader TeacherDashboardReader
}

// NewGetTeacherDashboardHandler creates a new GetTeacherDashboardHandler.
func NewGetTeacherDashboardHandler(reader TeacherDashboardReader) *GetTeacherDashboardHandler {
	return &GetTeacherDashboardHandler{reader: reader}
}

// Handle returns real-time teacher KPIs from MySQL.
func (h *GetTeacherDashboardHandler) Handle(ctx context.Context, instructorID, period string) (*TeacherDashboardResponse, error) {
	switch period {
	case "today", "7d", "30d", "year":
	default:
		period = "7d"
	}
	return h.reader.GetTeacherDashboard(ctx, instructorID, period)
}
