package classroom

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type StudentClassroomRouter struct {
	listStudentClassroomsHandler *queries.ListStudentClassroomsHandler
	enrollStudentHandler         *commands.EnrollStudentHandler
	contentPolicy                *access.ContentPolicy
}

func (r *StudentClassroomRouter) WithContentPolicy(policy *access.ContentPolicy) *StudentClassroomRouter {
	r.contentPolicy = policy
	return r
}

func NewStudentClassroomRouter(
	listStudentClassroomsHandler *queries.ListStudentClassroomsHandler,
	enrollStudentHandler *commands.EnrollStudentHandler,
) *StudentClassroomRouter {
	return &StudentClassroomRouter{
		listStudentClassroomsHandler: listStudentClassroomsHandler,
		enrollStudentHandler:         enrollStudentHandler,
	}
}

func (r *StudentClassroomRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	joinMiddlewares ...echo.MiddlewareFunc,
) {
	v1.GET("/student/classrooms", r.ListStudentClassrooms, authRequired)
	joinChain := append([]echo.MiddlewareFunc{authRequired}, joinMiddlewares...)
	v1.POST("/classrooms/join", r.JoinClassroom, joinChain...)
}

func requireStudentSession(c *echo.Context) (string, error) {
	userID, _ := c.Get("user_id").(string)
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	roleStr, _ := c.Get("user_role").(string)
	if entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr))) != entities.RoleStudent {
		return "", response.Fail(c, http.StatusForbidden, "Esta sección es exclusiva para estudiantes.")
	}
	return userID, nil
}

type JoinClassroomRequest struct {
	Code string `json:"code"`
}

func (r *StudentClassroomRouter) JoinClassroom(c *echo.Context) error {
	userID, errResp := requireStudentSession(c)
	if errResp != nil {
		return errResp
	}

	var req JoinClassroomRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	enrollment, err := r.enrollStudentHandler.Handle(c.Request().Context(), commands.EnrollStudentCommand{
		Code:      req.Code,
		StudentID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo unir al aula.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    enrollment,
		"message": "¡Te has unido al aula exitosamente!",
	})
}

func (r *StudentClassroomRouter) ListStudentClassrooms(c *echo.Context) error {
	userID, errResp := requireStudentSession(c)
	if errResp != nil {
		return errResp
	}

	classrooms, err := r.listStudentClassroomsHandler.Handle(c.Request().Context(), queries.ListStudentClassroomsQuery{
		StudentID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron obtener las aulas del estudiante.", err)
	}

	ctx := c.Request().Context()
	allowedByCourse := make(map[string]bool)
	payload := make([]map[string]any, 0, len(classrooms))
	for i := range classrooms {
		cls := &classrooms[i]
		for j := range cls.ResolvedCourses {
			course := &cls.ResolvedCourses[j]
			if course.Status != entities.ContentStatusPublished {
				continue
			}
			if _, seen := allowedByCourse[course.ID]; seen {
				continue
			}
			err := r.contentPolicy.Authorize(ctx, course, nil, userID, entities.RoleStudent)
			if err != nil && !errors.Is(err, access.ErrContentDenied) {
				return response.Error(c, http.StatusInternalServerError, "No se pudieron obtener las aulas del estudiante.", err)
			}
			allowedByCourse[course.ID] = err == nil
		}
		payload = append(payload, formatStudentClassroomDTO(cls, userID, allowedByCourse))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    payload,
	})
}

func formatStudentClassroomDTO(cls *entities.Classroom, studentID string, permissions ...map[string]bool) map[string]any {
	teacherName := "Docente Asignado"
	if cls.Teacher != nil && cls.Teacher.Name != "" {
		teacherName = cls.Teacher.Name
	}

	// Progreso del estudiante en esta cohorte
	wordsMastered := 0
	accuracyPct := 100
	streakDays := 0
	for _, enr := range cls.Enrollments {
		if enr.StudentID == studentID {
			wordsMastered = enr.WordsMastered
			accuracyPct = enr.AccuracyPct
			streakDays = enr.StreakDays
			break
		}
	}

	weeklyGoal := cls.WeeklyGoalWords
	if weeklyGoal <= 0 {
		weeklyGoal = 50
	}

	var accessByCourse map[string]bool
	if len(permissions) > 0 {
		accessByCourse = permissions[0]
	}

	// Cursos asignados con Pase Escolar
	assignedCourses := make([]map[string]any, 0, len(cls.ResolvedCourses))
	for _, course := range cls.ResolvedCourses {
		if course.Status != entities.ContentStatusPublished {
			continue
		}
		hasAccess := resolveClassroomCourseAccess(course, accessByCourse)
		lessons := studentCourseLessons(cls, course)
		lessonCount := len(lessons)
		assignedCourses = append(assignedCourses, map[string]any{
			"id":          course.ID,
			"title":       course.Title,
			"level":       course.Level,
			"description": course.Description,
			"lessonCount": lessonCount,
			"lessons":     lessons,
			"available":   true,
			"has_access":  hasAccess,
			"access_tier": course.AccessTier,
			"price_cents": course.PriceCents,
			"currency":    currencyOrPEN(course.Currency),
		})
	}

	// Minijuegos activos
	activeGamesList := make([]string, 0)
	if cls.ActiveGames != nil && *cls.ActiveGames != "" {
		for _, s := range strings.Split(*cls.ActiveGames, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				activeGamesList = append(activeGamesList, trimmed)
			}
		}
	}

	// Tropiezos colectivos (Stumbles)
	stumbles := make([]map[string]any, 0, len(cls.Stumbles))
	for _, st := range cls.Stumbles {
		stumbles = append(stumbles, map[string]any{
			"term":        st.Term,
			"translation": st.Translation,
			"failRate":    fmt.Sprintf("%d%%", st.FailRate),
			"fails":       st.TotalAttempts,
		})
	}

	teacherPayload := map[string]any{
		"id":   cls.TeacherID,
		"name": teacherName,
	}
	if cls.Teacher != nil {
		if cls.Teacher.AvatarURL != nil && *cls.Teacher.AvatarURL != "" {
			teacherPayload["avatar_url"] = *cls.Teacher.AvatarURL
		}
		if cls.Teacher.Specialty != nil && *cls.Teacher.Specialty != "" {
			teacherPayload["specialty"] = *cls.Teacher.Specialty
		}
		if cls.Teacher.Bio != nil && *cls.Teacher.Bio != "" {
			teacherPayload["bio"] = *cls.Teacher.Bio
		}
	}

	return map[string]any{
		"id":              cls.ID,
		"teacher_id":      cls.TeacherID,
		"name":            cls.Name,
		"code":            cls.Code,
		"level":           cls.Level,
		"description":     cls.Description,
		"weeklyGoalWords": weeklyGoal,
		"teacher":         teacherPayload,
		"studentProgress": map[string]any{
			"wordsMastered": wordsMastered,
			"weeklyGoal":    weeklyGoal,
			"accuracyPct":   accuracyPct,
			"streakDays":    streakDays,
		},
		"assignedCourses": assignedCourses,
		"activeGames":     activeGamesList,
		"stumbles":        stumbles,
	}
}

func resolveClassroomCourseAccess(course entities.Course, accessByCourse map[string]bool) bool {
	if accessByCourse != nil {
		return accessByCourse[course.ID]
	}
	return access.EffectiveOwnerType(&course) == entities.OwnerTypePlatform
}

func currencyOrPEN(currency string) string {
	if strings.TrimSpace(currency) == "" {
		return entities.CurrencyPEN
	}
	return strings.ToUpper(strings.TrimSpace(currency))
}
