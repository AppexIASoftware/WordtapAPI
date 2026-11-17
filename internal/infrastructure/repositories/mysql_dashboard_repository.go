package repositories

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// MySQLDashboardRepository aggregates live metrics and recent activity from MySQL.
type MySQLDashboardRepository struct {
	db *gorm.DB
}

// NewMySQLDashboardRepository creates a MySQL-backed DashboardReader.
func NewMySQLDashboardRepository(db *gorm.DB) *MySQLDashboardRepository {
	return &MySQLDashboardRepository{db: db}
}

type roleCountRow struct {
	Role  string
	Total int64
}

type tierCountRow struct {
	AccessTier string
	Total      int64
}

type statusCountRow struct {
	Status string
	Total  int64
}

func (r *MySQLDashboardRepository) GetAdminDashboard(ctx context.Context) (*dashQuery.AdminDashboardResponse, error) {
	res := &dashQuery.AdminDashboardResponse{
		RecentEvents: []dashQuery.DashboardEventItem{},
		GeneratedAt:  time.Now().UTC(),
	}

	if err := r.loadUserStats(ctx, &res.Users); err != nil {
		return nil, err
	}
	if err := r.loadCatalogStats(ctx, &res.Catalog); err != nil {
		return nil, err
	}
	events, err := r.loadRecentEvents(ctx)
	if err != nil {
		return nil, err
	}
	res.RecentEvents = events
	return res, nil
}

func (r *MySQLDashboardRepository) loadUserStats(ctx context.Context, out *dashQuery.DashboardUserStats) error {
	var roleRows []roleCountRow
	if err := r.db.WithContext(ctx).
		Model(&entities.User{}).
		Select("role, COUNT(*) as total").
		Group("role").
		Scan(&roleRows).Error; err != nil {
		return err
	}
	for _, row := range roleRows {
		out.TotalUsers += row.Total
		switch entities.UserRole(row.Role) {
		case entities.RoleStudent:
			out.Students = row.Total
		case entities.RoleInstructor:
			out.Instructors = row.Total
		case entities.RoleModerator:
			out.Moderators = row.Total
		case entities.RoleAdmin:
			out.Admins = row.Total
		}
	}

	var tierRows []tierCountRow
	if err := r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("role = ?", entities.RoleStudent).
		Select("access_tier, COUNT(*) as total").
		Group("access_tier").
		Scan(&tierRows).Error; err != nil {
		return err
	}
	for _, row := range tierRows {
		switch entities.AccessTier(row.AccessTier) {
		case entities.AccessTierFree:
			out.FreeStudents = row.Total
		case entities.AccessTierCourse:
			out.CourseStudents = row.Total
		case entities.AccessTierSubscription:
			out.SubStudents = row.Total
		}
	}

	var courseAccessCount int64
	if err := r.db.WithContext(ctx).
		Table("users u").
		Joins("LEFT JOIN course_access ca ON ca.user_id = u.id AND ca.status = ?", "active").
		Where("u.role = ? AND (u.access_tier = ? OR ca.id IS NOT NULL)", entities.RoleStudent, entities.AccessTierCourse).
		Distinct("u.id").
		Count(&courseAccessCount).Error; err == nil {
		out.CourseStudents = courseAccessCount
	}
	return nil
}

func (r *MySQLDashboardRepository) loadCatalogStats(ctx context.Context, out *dashQuery.DashboardCatalogStats) error {
	var courseRows []statusCountRow
	if err := r.db.WithContext(ctx).
		Model(&entities.Course{}).
		Select("status, COUNT(*) as total").
		Group("status").
		Scan(&courseRows).Error; err != nil {
		return err
	}
	for _, row := range courseRows {
		out.TotalCourses += row.Total
		switch entities.ContentStatus(row.Status) {
		case entities.ContentStatusPublished:
			out.PublishedCourses = row.Total
		case entities.ContentStatusInReview:
			out.InReviewCourses = row.Total
		case entities.ContentStatusDraft:
			out.DraftCourses = row.Total
		}
	}

	var appRows []statusCountRow
	if err := r.db.WithContext(ctx).
		Model(&entities.TeacherApplication{}).
		Select("status, COUNT(*) as total").
		Group("status").
		Scan(&appRows).Error; err != nil {
		return err
	}
	for _, row := range appRows {
		switch entities.ApplicationStatus(row.Status) {
		case entities.ApplicationStatusPending:
			out.PendingTeacherApps = row.Total
		case entities.ApplicationStatusApproved:
			out.ApprovedTeacherApps = row.Total
		}
	}

	if err := r.db.WithContext(ctx).Model(&entities.ContentBankItem{}).Count(&out.TotalVaultItems).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&entities.TeacherGame{}).Count(&out.TotalGames).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&entities.Classroom{}).Count(&out.TotalClassrooms).Error; err != nil {
		return err
	}
	return nil
}

func (r *MySQLDashboardRepository) loadRecentEvents(ctx context.Context) ([]dashQuery.DashboardEventItem, error) {
	var events []dashQuery.DashboardEventItem

	var auditRows []entities.AuditLog
	if err := r.db.WithContext(ctx).
		Preload("ActorUser").
		Order("created_at DESC").
		Limit(5).
		Find(&auditRows).Error; err == nil {
		for _, a := range auditRows {
			actor := "Sistema"
			if a.ActorUser != nil && a.ActorUser.Name != "" {
				actor = a.ActorUser.Name
			}
			events = append(events, dashQuery.DashboardEventItem{
				ID:        "audit-" + a.ID,
				Action:    a.Action,
				Summary:   fmt.Sprintf("%s actualizó %s", actor, a.EntityType),
				Actor:     actor,
				CreatedAt: a.CreatedAt,
			})
		}
	}

	var apps []entities.TeacherApplication
	if err := r.db.WithContext(ctx).
		Preload("User").
		Order("updated_at DESC").
		Limit(4).
		Find(&apps).Error; err == nil {
		for _, app := range apps {
			name := "Docente"
			if app.User != nil && app.User.Name != "" {
				name = app.User.Name
			}
			events = append(events, dashQuery.DashboardEventItem{
				ID:        "app-" + app.ID,
				Action:    "teacher.application." + string(app.Status),
				Summary:   fmt.Sprintf("Postulación docente de %s (%s)", name, app.Status),
				Actor:     name,
				CreatedAt: app.UpdatedAt,
			})
		}
	}

	var courses []entities.Course
	if err := r.db.WithContext(ctx).
		Order("updated_at DESC").
		Limit(4).
		Find(&courses).Error; err == nil {
		for _, c := range courses {
			events = append(events, dashQuery.DashboardEventItem{
				ID:        "course-" + c.ID,
				Action:    "course." + string(c.Status),
				Summary:   fmt.Sprintf("Curso '%s' en estado %s", c.Title, c.Status),
				Actor:     "Catálogo",
				CreatedAt: c.UpdatedAt,
			})
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
	if len(events) > 6 {
		events = events[:6]
	}
	return events, nil
}

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

// GetTeacherDashboard aggregates real student, vault, and revenue metrics from MySQL.
func (r *MySQLDashboardRepository) GetTeacherDashboard(ctx context.Context, instructorID, period string) (*dashQuery.TeacherDashboardResponse, error) {
	now := time.Now().UTC()
	since := teacherPeriodStart(period, now)

	res := &dashQuery.TeacherDashboardResponse{
		Period:      period,
		GeneratedAt: now,
	}

	r.db.WithContext(ctx).
		Model(&entities.UserDailyActivity{}).
		Where("activity_date >= ?", since).
		Distinct("user_id").
		Count(&res.ActiveStudents)

	var avgStreak struct {
		Avg float64
	}
	r.db.WithContext(ctx).
		Model(&entities.UserLearningStats{}).
		Select("COALESCE(AVG(current_streak_days), 0) as avg").
		Scan(&avgStreak)
	res.AvgStreakDays = avgStreak.Avg

	r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("role = ?", entities.RoleStudent).
		Count(&res.TotalStudents)

	if res.TotalStudents > 0 {
		var retained int64
		r.db.WithContext(ctx).
			Model(&entities.UserLearningStats{}).
			Where("last_activity_date >= ?", since).
			Count(&retained)
		res.RetentionPercent = float64(retained) * 100 / float64(res.TotalStudents)
	}

	vaultQuery := r.db.WithContext(ctx).Model(&entities.ContentBankItem{})
	if instructorID != "" {
		vaultQuery = vaultQuery.Where("is_official = ? OR created_by = ?", true, instructorID)
	} else {
		vaultQuery = vaultQuery.Where("is_official = ?", true)
	}
	vaultQuery.Count(&res.VaultItems)
	r.db.WithContext(ctx).Model(&entities.ContentBankType{}).Count(&res.VaultBanks)

	var rev struct {
		TotalCents int64
		Count      int64
	}
	r.db.WithContext(ctx).
		Model(&entities.Order{}).
		Where("status = ? AND created_at >= ?", entities.PaymentStatusApproved, since).
		Select("COALESCE(SUM(amount_cents), 0) as total_cents, COUNT(*) as count").
		Scan(&rev)
	res.RevenueCents = rev.TotalCents
	res.PaidOrders = rev.Count

	r.db.WithContext(ctx).
		Model(&entities.Subscription{}).
		Where("status = ?", "active").
		Count(&res.ActiveSubscriptions)

	return res, nil
}
