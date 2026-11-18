package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func teacherPeriodStart(period string, now time.Time) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch period {
	case "today":
		return today
	case "30d":
		return today.AddDate(0, 0, -29)
	case "year":
		return today.AddDate(-1, 0, 0)
	default:
		return today.AddDate(0, 0, -6)
	}
}

func scopeToInstructorStudents(q *gorm.DB, userIDCol, instructorID string) *gorm.DB {
	if instructorID == "" {
		return q
	}
	var allowedCol string
	switch userIDCol {
	case "u.id", "uda.user_id", "users.id":
		allowedCol = userIDCol
	default:
		return q.Where("1 = 0")
	}
	clause := fmt.Sprintf(
		"(EXISTS (SELECT 1 FROM classroom_enrollments ce INNER JOIN classrooms cl ON cl.id = ce.classroom_id WHERE ce.student_id = %s AND cl.teacher_id = ?) OR EXISTS (SELECT 1 FROM user_lesson_progresses ulp INNER JOIN lessons l ON l.id = ulp.lesson_id INNER JOIN courses c ON c.id = l.course_id WHERE ulp.user_id = %s AND c.created_by = ? AND c.owner_type = ?))",
		allowedCol, allowedCol,
	)
	return q.Where(clause, instructorID, instructorID, entities.OwnerTypeTeacher)
}

// GetTeacherDashboard aggregates tenant-scoped student, vault, and revenue metrics from MySQL in 4 SQL round-trips.
func (r *MySQLDashboardRepository) GetTeacherDashboard(ctx context.Context, instructorID, period string) (*dashQuery.TeacherDashboardResponse, error) {
	now := time.Now().UTC()
	since := teacherPeriodStart(period, now)
	instructorID = strings.TrimSpace(instructorID)

	res := &dashQuery.TeacherDashboardResponse{
		Period:      period,
		GeneratedAt: now,
	}

	var studentKPIs struct {
		TotalStudents int64
		AvgStreak     float64
		Retained      int64
	}
	studentQ := scopeToInstructorStudents(
		r.db.WithContext(ctx).
			Table("users u").
			Joins("LEFT JOIN user_learning_stats uls ON uls.user_id = u.id").
			Where("u.role = ?", entities.RoleStudent),
		"u.id",
		instructorID,
	)
	if err := studentQ.Select(
		"COUNT(u.id) AS total_students, COALESCE(AVG(uls.current_streak_days), 0) AS avg_streak, COALESCE(SUM(CASE WHEN uls.last_activity_date >= ? THEN 1 ELSE 0 END), 0) AS retained",
		since,
	).Scan(&studentKPIs).Error; err != nil {
		return nil, err
	}
	res.TotalStudents = studentKPIs.TotalStudents
	res.AvgStreakDays = studentKPIs.AvgStreak
	if res.TotalStudents > 0 {
		res.RetentionPercent = float64(studentKPIs.Retained) * 100 / float64(res.TotalStudents)
	}

	activeQ := scopeToInstructorStudents(
		r.db.WithContext(ctx).
			Table("user_daily_activities uda").
			Joins("INNER JOIN users u ON u.id = uda.user_id AND u.role = ?", entities.RoleStudent).
			Where("uda.activity_date >= ?", since),
		"uda.user_id",
		instructorID,
	)
	if err := activeQ.Distinct("uda.user_id").Count(&res.ActiveStudents).Error; err != nil {
		return nil, err
	}

	var vaultKPIs struct {
		VaultItems int64
		VaultBanks int64
	}
	vaultSQL := "SELECT (SELECT COUNT(*) FROM content_bank_items WHERE created_by IS NULL) AS vault_items, (SELECT COUNT(*) FROM content_bank_types WHERE is_active = ?) AS vault_banks"
	vaultArgs := []any{true}
	if instructorID != "" {
		vaultSQL = "SELECT (SELECT COUNT(*) FROM content_bank_items WHERE created_by IS NULL OR created_by = ?) AS vault_items, (SELECT COUNT(*) FROM content_bank_types WHERE is_active = ?) AS vault_banks"
		vaultArgs = []any{instructorID, true}
	}
	if err := r.db.WithContext(ctx).Raw(vaultSQL, vaultArgs...).Scan(&vaultKPIs).Error; err != nil {
		return nil, err
	}
	res.VaultItems = vaultKPIs.VaultItems
	res.VaultBanks = vaultKPIs.VaultBanks

	if instructorID == "" {
		var adminBilling struct {
			TotalCents int64
			Count      int64
			ActiveSubs int64
		}
		if err := r.db.WithContext(ctx).Raw(
			"SELECT COALESCE(SUM(amount_cents), 0) AS total_cents, COUNT(*) AS count, (SELECT COUNT(*) FROM subscriptions WHERE status = ?) AS active_subs FROM orders WHERE status = ? AND created_at >= ?",
			"active", entities.PaymentStatusApproved, since,
		).Scan(&adminBilling).Error; err != nil {
			return nil, err
		}
		res.RevenueCents = adminBilling.TotalCents
		res.PaidOrders = adminBilling.Count
		res.ActiveSubscriptions = adminBilling.ActiveSubs
	} else if r.simulationEnabled {
		var rev struct {
			TotalCents int64
			Count      int64
		}
		// Optional simulation table is not in AutoMigrate; query best-effort scoped to teacher-owned courses.
		if err := r.db.WithContext(ctx).
			Table("simulated_course_purchases scp").
			Joins("INNER JOIN courses c ON c.id = scp.course_id").
			Where("c.created_by = ? AND c.owner_type = ? AND scp.created_at >= ?", instructorID, entities.OwnerTypeTeacher, since).
			Select("COALESCE(SUM(scp.amount_cents), 0) as total_cents, COUNT(*) as count").
			Scan(&rev).Error; err == nil {
			res.RevenueCents = rev.TotalCents
			res.PaidOrders = rev.Count
		}
	}

	return res, nil
}
