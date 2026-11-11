package classroom

import (
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// splitCSV turns a comma separated column value into a trimmed list.
func splitCSV(value *string) []string {
	list := make([]string, 0)
	if value == nil || *value == "" {
		return list
	}
	for _, raw := range strings.Split(*value, ",") {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return list
}

// classroomCourseIDs resolves the assigned courses, falling back to the single
// legacy course reference when the classroom has no explicit selection.
func classroomCourseIDs(classroom *entities.Classroom) []string {
	if list := splitCSV(classroom.AssignedCourses); len(list) > 0 {
		return list
	}
	if courseID := classroomCourseID(classroom); courseID != "" {
		return []string{courseID}
	}
	return []string{}
}

func classroomCourseTitle(classroom *entities.Classroom, fallback string) string {
	if classroom.Course != nil {
		return classroom.Course.Title
	}
	return fallback
}

func classroomCourseID(classroom *entities.Classroom) string {
	if classroom.Course != nil {
		return classroom.Course.ID
	}
	if classroom.CourseID != nil {
		return *classroom.CourseID
	}
	return ""
}

func formatClassroomDTO(classroom *entities.Classroom) map[string]any {
	return map[string]any{
		"id":              classroom.ID,
		"name":            classroom.Name,
		"code":            classroom.Code,
		"courseId":        classroomCourseID(classroom),
		"courseTitle":     classroomCourseTitle(classroom, "Curso Asignado"),
		"assignedCourses": classroomCourseIDs(classroom),
		"assignedLessons": classroomLessonIDs(classroom),
		"activeGames":     splitCSV(classroom.ActiveGames),
		"weeklyGoalWords": classroom.WeeklyGoalWords,
		"level":           classroom.Level,
		"created":         classroom.CreatedAt.Format("02/01/2006"),
		"studentsCount":   len(classroom.Enrollments),
		"stats": map[string]any{
			"sub":     0,
			"trial":   0,
			"free":    0,
			"scholar": 0,
		},
		"stumble":  []map[string]any{},
		"students": []map[string]any{},
	}
}

func studentInitials(name string) string {
	parts := strings.Split(name, " ")
	if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
		return strings.ToUpper(string(parts[0][0]) + string(parts[1][0]))
	}
	if len(parts) == 1 && len(parts[0]) > 0 {
		return strings.ToUpper(string(parts[0][:1]))
	}
	return "ST"
}

// formatEnrolledStudents maps the classroom roster and counts each access tier.
func formatEnrolledStudents(classroom *entities.Classroom) ([]map[string]any, int, int) {
	students := make([]map[string]any, 0, len(classroom.Enrollments))
	subCount := 0
	freeCount := 0

	for _, enr := range classroom.Enrollments {
		name := "Estudiante"
		email := ""
		initials := "ST"
		payStatus := "free"
		payLabel := "Plan Gratuito"

		if enr.Student != nil {
			name = enr.Student.Name
			email = enr.Student.Email
			initials = studentInitials(name)

			switch enr.Student.AccessTier {
			case entities.AccessTierSubscription:
				payStatus = "subscription"
				payLabel = "Suscripción Activa"
				subCount++
			case entities.AccessTierCourse:
				payStatus = "course_purchased"
				payLabel = "Curso Adquirido"
				subCount++
			default:
				payStatus = "free"
				payLabel = "Plan Gratuito"
				freeCount++
			}
		}

		students = append(students, map[string]any{
			"id":          enr.ID,
			"name":        name,
			"email":       email,
			"initials":    initials,
			"avatarColor": "emerald",
			"level":       classroom.Level,
			"streak":      fmt.Sprintf("%d días", enr.StreakDays),
			"words":       enr.WordsMastered,
			"accuracy":    fmt.Sprintf("%d%%", enr.AccuracyPct),
			"enrolledAt":  enr.EnrolledAt.Format("02 Jan 2006"),
			"lastActive":  "Hoy",
			"payStatus":   payStatus,
			"payLabel":    payLabel,
			"renewDate":   "Renueva próx. mes",
		})
	}

	return students, subCount, freeCount
}

func formatStumbles(classroom *entities.Classroom) []map[string]any {
	stumbles := make([]map[string]any, 0, len(classroom.Stumbles))
	for _, st := range classroom.Stumbles {
		desc := st.Translation
		if st.Notes != nil && *st.Notes != "" {
			desc = *st.Notes
		}
		stumbles = append(stumbles, map[string]any{
			"term":     st.Term,
			"failRate": fmt.Sprintf("%d%%", st.FailRate),
			"desc":     desc,
		})
	}
	return stumbles
}

// formatClassroomListDTO builds the teacher roster payload for one classroom.
func formatClassroomListDTO(classroom *entities.Classroom) map[string]any {
	students, subCount, freeCount := formatEnrolledStudents(classroom)

	weeklyGoal := classroom.WeeklyGoalWords
	if weeklyGoal <= 0 {
		weeklyGoal = 50
	}

	return map[string]any{
		"id":              classroom.ID,
		"name":            classroom.Name,
		"code":            classroom.Code,
		"courseId":        classroomCourseID(classroom),
		"courseTitle":     classroomCourseTitle(classroom, "Curso General"),
		"assignedCourses": classroomCourseIDs(classroom),
		"assignedLessons": classroomLessonIDs(classroom),
		"activeGames":     splitCSV(classroom.ActiveGames),
		"weeklyGoalWords": weeklyGoal,
		"level":           classroom.Level,
		"created":         classroom.CreatedAt.Format("02/01/2006"),
		"studentsCount":   len(classroom.Enrollments),
		"stats": map[string]any{
			"sub":     subCount,
			"trial":   0,
			"free":    freeCount,
			"scholar": 0,
		},
		"stumble":  formatStumbles(classroom),
		"students": students,
	}
}
