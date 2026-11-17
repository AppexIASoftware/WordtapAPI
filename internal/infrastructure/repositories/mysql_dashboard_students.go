package repositories

import (
	"context"
	"strings"
	"time"

	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// ListGlobalStudents returns paginated students with learning stats, plan tier, and latest course progress.
// When filter.InstructorID is set, rows, courses, and progress are scoped strictly to that instructor's classrooms and courses.
func (r *MySQLDashboardRepository) ListGlobalStudents(ctx context.Context, filter dashQuery.GlobalStudentFilter) (*dashQuery.GlobalStudentsResponse, error) {
	q := r.db.WithContext(ctx).Model(&entities.User{}).Where("users.role = ?", entities.RoleStudent)

	instructorID := strings.TrimSpace(filter.InstructorID)
	if instructorID != "" {
		q = q.Where(
			"(EXISTS (SELECT 1 FROM classroom_enrollments ce INNER JOIN classrooms cl ON cl.id = ce.classroom_id WHERE ce.student_id = users.id AND cl.teacher_id = ?) OR EXISTS (SELECT 1 FROM user_lesson_progresses ulp INNER JOIN lessons l ON l.id = ulp.lesson_id INNER JOIN courses c ON c.id = l.course_id WHERE ulp.user_id = users.id AND c.created_by = ?))",
			instructorID, instructorID,
		)
	}

	if plan := strings.TrimSpace(filter.Plan); plan != "" && plan != "all" {
		q = q.Where("users.access_tier = ?", plan)
	}
	if status := strings.TrimSpace(filter.Status); status == "active" {
		q = q.Where("users.is_active = ?", true)
	} else if status == "suspended" {
		q = q.Where("users.is_active = ?", false)
	}
	if level := strings.TrimSpace(filter.Level); level != "" && level != "all" {
		q = q.Where("LOWER(users.learning_level) LIKE ?", "%"+strings.ToLower(level)+"%")
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where("(LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?)", like, like)
	}
	if courseID := strings.TrimSpace(filter.CourseID); courseID != "" && courseID != "all" {
		q = q.Where(
			"(EXISTS (SELECT 1 FROM user_lesson_progresses ulp INNER JOIN lessons l ON l.id = ulp.lesson_id WHERE ulp.user_id = users.id AND l.course_id = ?) OR EXISTS (SELECT 1 FROM classroom_enrollments ce INNER JOIN classrooms c ON c.id = ce.classroom_id WHERE ce.student_id = users.id AND c.course_id = ?))",
			courseID, courseID,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var users []entities.User
	if err := q.Preload("LearningStats").
		Order("users.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&users).Error; err != nil {
		return nil, err
	}

	var courseRows []dashQuery.GlobalCourseOption
	courseQuery := r.db.WithContext(ctx).
		Model(&entities.Course{}).
		Select("id, title")
	if instructorID != "" {
		courseQuery = courseQuery.Where("status = ? AND created_by = ?", entities.ContentStatusPublished, instructorID)
	} else {
		courseQuery = courseQuery.Where("status = ? AND owner_type = ?", entities.ContentStatusPublished, entities.OwnerTypePlatform)
	}
	courseQuery.Order("title ASC").Scan(&courseRows)

	userIDs := make([]string, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	latestCourseByUser := make(map[string]dashQuery.GlobalCourseOption, len(userIDs))
	completedByUserCourse := make(map[string]int)
	totalLessonsByCourse := make(map[string]int)

	if len(userIDs) > 0 {
		var progressRows []struct {
			UserID      string
			CourseID    string
			CourseTitle string
			UpdatedAt   time.Time
		}
		progressQuery := r.db.WithContext(ctx).
			Table("user_lesson_progresses ulp").
			Select("ulp.user_id, c.id as course_id, c.title as course_title, ulp.updated_at").
			Joins("INNER JOIN lessons l ON l.id = ulp.lesson_id").
			Joins("INNER JOIN courses c ON c.id = l.course_id").
			Where("ulp.user_id IN ?", userIDs)
		if instructorID != "" {
			progressQuery = progressQuery.Where("c.created_by = ?", instructorID)
		}
		progressQuery.Order("ulp.updated_at DESC").Scan(&progressRows)

		courseIDSet := make(map[string]struct{})
		for _, pr := range progressRows {
			if _, ok := latestCourseByUser[pr.UserID]; !ok {
				latestCourseByUser[pr.UserID] = dashQuery.GlobalCourseOption{
					ID:    pr.CourseID,
					Title: pr.CourseTitle,
				}
			}
			completedByUserCourse[pr.UserID+":"+pr.CourseID]++
			courseIDSet[pr.CourseID] = struct{}{}
		}

		if len(courseIDSet) > 0 {
			courseIDs := make([]string, 0, len(courseIDSet))
			for cid := range courseIDSet {
				courseIDs = append(courseIDs, cid)
			}
			var lessonCounts []struct {
				CourseID string
				Total    int
			}
			r.db.WithContext(ctx).
				Table("lessons").
				Select("course_id, COUNT(*) as total").
				Where("course_id IN ? AND status = ?", courseIDs, entities.ContentStatusPublished).
				Group("course_id").
				Scan(&lessonCounts)
			for _, lc := range lessonCounts {
				totalLessonsByCourse[lc.CourseID] = lc.Total
			}
		}
	}

	items := make([]dashQuery.GlobalStudentItem, 0, len(users))
	for _, u := range users {
		streak := 0
		var lastAct *time.Time
		lessonsDone := 0
		if u.LearningStats != nil {
			streak = u.LearningStats.CurrentStreakDays
			lastAct = u.LearningStats.LastActivityDate
			lessonsDone = u.LearningStats.LessonsCompleted
		}

		courseInfo := latestCourseByUser[u.ID]
		if courseInfo.Title == "" {
			if instructorID != "" {
				courseInfo.Title = "Tus Cursos y Aulas"
			} else {
				courseInfo.Title = "Catálogo General"
			}
		}

		progress := 0
		if courseInfo.ID != "" && totalLessonsByCourse[courseInfo.ID] > 0 {
			done := completedByUserCourse[u.ID+":"+courseInfo.ID]
			progress = (done * 100) / totalLessonsByCourse[courseInfo.ID]
		} else if instructorID == "" {
			progress = lessonsDone * 15
		}
		if progress > 100 {
			progress = 100
		}

		items = append(items, dashQuery.GlobalStudentItem{
			ID:                u.ID,
			Name:              u.Name,
			Email:             u.Email,
			LearningLevel:     u.LearningLevel,
			CourseID:          courseInfo.ID,
			CourseTitle:       courseInfo.Title,
			CurrentStreakDays: streak,
			ProgressPercent:   progress,
			AccessTier:        string(u.AccessTier),
			IsActive:          u.IsActive,
			CreatedAt:         u.CreatedAt,
			LastActivityDate:  lastAct,
		})
	}

	return &dashQuery.GlobalStudentsResponse{
		Items:   items,
		Total:   total,
		Courses: courseRows,
	}, nil
}
