package classroom

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type ClassroomRouter struct {
	createClassroomHandler *commands.CreateClassroomHandler
	updateClassroomHandler *commands.UpdateClassroomHandler
	enrollStudentHandler   *commands.EnrollStudentHandler
	listClassroomsHandler  *queries.ListTeacherClassroomsHandler
	getStumbleListHandler  *queries.GetStumbleListHandler
}

func NewClassroomRouter(
	createClassroomHandler *commands.CreateClassroomHandler,
	updateClassroomHandler *commands.UpdateClassroomHandler,
	enrollStudentHandler *commands.EnrollStudentHandler,
	listClassroomsHandler *queries.ListTeacherClassroomsHandler,
	getStumbleListHandler *queries.GetStumbleListHandler,
) *ClassroomRouter {
	return &ClassroomRouter{
		createClassroomHandler: createClassroomHandler,
		updateClassroomHandler: updateClassroomHandler,
		enrollStudentHandler:   enrollStudentHandler,
		listClassroomsHandler:  listClassroomsHandler,
		getStumbleListHandler:  getStumbleListHandler,
	}
}

func (r *ClassroomRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	// Rutas de administración de aulas para docentes
	teacherGroup := v1.Group("/teacher/classrooms")
	teacherGroup.Use(authRequired, instructorOrAdmin)
	teacherGroup.POST("", r.CreateClassroom)
	teacherGroup.PUT("/:id", r.UpdateClassroom)
	teacherGroup.GET("", r.ListTeacherClassrooms)
	teacherGroup.GET("/:id/stumbles", r.GetClassroomStumbles)

	// Ruta de inscripción para estudiantes mediante código
	v1.POST("/classrooms/join", r.JoinClassroom, authRequired)
}

type CreateClassroomRequest struct {
	Name            string   `json:"name"`
	Level           string   `json:"level"`
	CourseID        *string  `json:"course_id"`
	CourseIDs       []string `json:"course_ids"`
	GameIDs         []string `json:"game_ids"`
	WeeklyGoalWords int      `json:"weekly_goal_words"`
	Description     *string  `json:"description"`
}

type UpdateClassroomRequest struct {
	Name            string   `json:"name"`
	Level           string   `json:"level"`
	CourseID        *string  `json:"course_id"`
	CourseIDs       []string `json:"course_ids"`
	GameIDs         []string `json:"game_ids"`
	WeeklyGoalWords int      `json:"weekly_goal_words"`
	Description     *string  `json:"description"`
}

func (r *ClassroomRouter) CreateClassroom(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	}

	var req CreateClassroomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	cmd := commands.CreateClassroomCommand{
		TeacherID:       userID,
		Name:            req.Name,
		Level:           req.Level,
		CourseID:        req.CourseID,
		CourseIDs:       req.CourseIDs,
		GameIDs:         req.GameIDs,
		WeeklyGoalWords: req.WeeklyGoalWords,
		Description:     req.Description,
	}

	classroom, err := r.createClassroomHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to create classroom",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"success": true,
		"data":    formatClassroomDTO(classroom),
	})
}

func (r *ClassroomRouter) UpdateClassroom(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	}

	classroomID := c.Param("id")
	if classroomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "classroom id required"})
	}

	var req UpdateClassroomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	cmd := commands.UpdateClassroomCommand{
		ClassroomID:     classroomID,
		TeacherID:       userID,
		Name:            req.Name,
		Level:           req.Level,
		CourseID:        req.CourseID,
		CourseIDs:       req.CourseIDs,
		GameIDs:         req.GameIDs,
		WeeklyGoalWords: req.WeeklyGoalWords,
		Description:     req.Description,
	}

	classroom, err := r.updateClassroomHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to update classroom",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    formatClassroomDTO(classroom),
	})
}

func formatClassroomDTO(classroom *entities.Classroom) map[string]any {
	courseTitle := "Curso Asignado"
	courseID := ""
	if classroom.Course != nil {
		courseTitle = classroom.Course.Title
		courseID = classroom.Course.ID
	} else if classroom.CourseID != nil {
		courseID = *classroom.CourseID
	}

	assignedCoursesList := make([]string, 0)
	if classroom.AssignedCourses != nil && *classroom.AssignedCourses != "" {
		for _, s := range strings.Split(*classroom.AssignedCourses, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				assignedCoursesList = append(assignedCoursesList, trimmed)
			}
		}
	} else if courseID != "" {
		assignedCoursesList = append(assignedCoursesList, courseID)
	}

	activeGamesList := make([]string, 0)
	if classroom.ActiveGames != nil && *classroom.ActiveGames != "" {
		for _, s := range strings.Split(*classroom.ActiveGames, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				activeGamesList = append(activeGamesList, trimmed)
			}
		}
	}

	return map[string]any{
		"id":              classroom.ID,
		"name":            classroom.Name,
		"code":            classroom.Code,
		"courseId":        courseID,
		"courseTitle":     courseTitle,
		"assignedCourses": assignedCoursesList,
		"activeGames":     activeGamesList,
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

func (r *ClassroomRouter) ListTeacherClassrooms(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	}

	classrooms, err := r.listClassroomsHandler.Handle(c.Request().Context(), queries.ListTeacherClassroomsQuery{
		TeacherID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to list classrooms",
			"details": err.Error(),
		})
	}

	response := make([]map[string]any, 0, len(classrooms))
	for _, cls := range classrooms {
		// Mapear estudiantes
		studentsList := make([]map[string]any, 0, len(cls.Enrollments))
		subCount := 0
		trialCount := 0
		freeCount := 0

		for _, enr := range cls.Enrollments {
			name := "Estudiante"
			email := ""
			initials := "ST"
			payStatus := "free"
			payLabel := "Plan Gratuito"

			if enr.Student != nil {
				name = enr.Student.Name
				email = enr.Student.Email
				parts := strings.Split(name, " ")
				if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
					initials = strings.ToUpper(string(parts[0][0]) + string(parts[1][0]))
				} else if len(parts) == 1 && len(parts[0]) > 0 {
					initials = strings.ToUpper(string(parts[0][:1]))
				}

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

			streakStr := fmt.Sprintf("%d días", enr.StreakDays)
			accuracyStr := fmt.Sprintf("%d%%", enr.AccuracyPct)

			studentsList = append(studentsList, map[string]any{
				"id":           enr.ID,
				"name":         name,
				"email":        email,
				"initials":     initials,
				"avatarColor":  "emerald",
				"level":        cls.Level,
				"streak":       streakStr,
				"words":        enr.WordsMastered,
				"accuracy":     accuracyStr,
				"enrolledAt":   enr.EnrolledAt.Format("02 Jan 2006"),
				"lastActive":   "Hoy",
				"payStatus":    payStatus,
				"payLabel":     payLabel,
				"renewDate":    "Renueva próx. mes",
			})
		}

		// Mapear tropiezos
		stumbleList := make([]map[string]any, 0, len(cls.Stumbles))
		for _, st := range cls.Stumbles {
			desc := st.Translation
			if st.Notes != nil && *st.Notes != "" {
				desc = *st.Notes
			}
			stumbleList = append(stumbleList, map[string]any{
				"term":     st.Term,
				"failRate": fmt.Sprintf("%d%%", st.FailRate),
				"desc":     desc,
			})
		}

		courseTitle := "Curso General"
		courseID := ""
		if cls.Course != nil {
			courseTitle = cls.Course.Title
			courseID = cls.Course.ID
		} else if cls.CourseID != nil {
			courseID = *cls.CourseID
		}

		assignedCoursesList := make([]string, 0)
		if cls.AssignedCourses != nil && *cls.AssignedCourses != "" {
			for _, s := range strings.Split(*cls.AssignedCourses, ",") {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					assignedCoursesList = append(assignedCoursesList, trimmed)
				}
			}
		} else if courseID != "" {
			assignedCoursesList = append(assignedCoursesList, courseID)
		}

		activeGamesList := make([]string, 0)
		if cls.ActiveGames != nil && *cls.ActiveGames != "" {
			for _, s := range strings.Split(*cls.ActiveGames, ",") {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					activeGamesList = append(activeGamesList, trimmed)
				}
			}
		}

		weeklyGoal := cls.WeeklyGoalWords
		if weeklyGoal <= 0 {
			weeklyGoal = 50
		}

		response = append(response, map[string]any{
			"id":              cls.ID,
			"name":            cls.Name,
			"code":            cls.Code,
			"courseId":        courseID,
			"courseTitle":     courseTitle,
			"assignedCourses": assignedCoursesList,
			"activeGames":     activeGamesList,
			"weeklyGoalWords": weeklyGoal,
			"level":           cls.Level,
			"created":         cls.CreatedAt.Format("02/01/2006"),
			"studentsCount":   len(cls.Enrollments),
			"stats": map[string]any{
				"sub":     subCount,
				"trial":   trialCount,
				"free":    freeCount,
				"scholar": 0,
			},
			"stumble":  stumbleList,
			"students": studentsList,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    response,
	})
}

func (r *ClassroomRouter) GetClassroomStumbles(c *echo.Context) error {
	classroomID := c.Param("id")
	if classroomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "classroom id required"})
	}

	stumbles, err := r.getStumbleListHandler.Handle(c.Request().Context(), queries.GetStumbleListQuery{
		ClassroomID: classroomID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to get stumble list",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    stumbles,
	})
}

type JoinClassroomRequest struct {
	Code string `json:"code"`
}

func (r *ClassroomRouter) JoinClassroom(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	}

	var req JoinClassroomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	enrollment, err := r.enrollStudentHandler.Handle(c.Request().Context(), commands.EnrollStudentCommand{
		Code:      req.Code,
		StudentID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to join classroom",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    enrollment,
		"message": "¡Te has unido al aula exitosamente!",
	})
}
