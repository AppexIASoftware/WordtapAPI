package rest

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"

	classroomCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	classroomQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	catalogQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/catalog/queries"
	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	gameCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	gameQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	settingQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/queries"
	teacherAppCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/commands"
	teacherAppQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	vaultCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	vaultQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/auth"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/catalog"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/course"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/docs"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/health"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/lesson"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/setting"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/teacher_application"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vault"
)

type Server struct {
	App *echo.Echo
}

func NewServer(db *gorm.DB) (*Server, error) {
	// 1. Validar variables de entorno requeridas
	corsOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if corsOrigins == "" {
		return nil, fmt.Errorf("missing required environment variable: CORS_ALLOWED_ORIGINS")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return nil, fmt.Errorf("missing required environment variable: JWT_SECRET")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes long")
	}

	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if googleClientID == "" {
		return nil, fmt.Errorf("missing required environment variable: GOOGLE_CLIENT_ID")
	}

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	devLoginEnabled := appEnv == "development" || appEnv == "dev" || appEnv == "local" || appEnv == ""

	e := echo.New()

	// Middlewares globales
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	var origins []string
	for _, o := range strings.Split(corsOrigins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one valid origin")
	}
	e.Use(middleware.CORS(origins...))

	// Configuración de seguridad y servicios auxiliares
	jwtService := security.NewJWTService(jwtSecret, 15*time.Minute, 30*24*time.Hour)
	googleVerifier := security.NewHTTPGoogleVerifier(googleClientID)

	// Repositorios de infraestructura
	lessonRepo := infraRepo.NewMySQLLessonRepository(db)
	userRepo := infraRepo.NewMySQLUserRepository(db)
	courseRepo := infraRepo.NewMySQLCourseRepository(db)
	courseReviewRepo := infraRepo.NewMySQLCourseReviewRepository(db)
	teacherAppRepo := infraRepo.NewMySQLTeacherApplicationRepository(db)
	settingRepo := infraRepo.NewMySQLPlatformSettingRepository(db)
	contentBankRepo := infraRepo.NewMySQLContentBankRepository(db)
	classroomRepo := infraRepo.NewMySQLClassroomRepository(db)
	teacherGameRepo := infraRepo.NewMySQLTeacherGameRepository(db)
	lessonGameRepo := infraRepo.NewMySQLLessonGameRepository(db)

	// Casos de uso
	loginWithGoogleHandler := commands.NewLoginWithGoogleHandler(userRepo, googleVerifier, jwtService, teacherAppRepo)
	// Classroom Command & Query handlers
	createClassroomHandler := classroomCmd.NewCreateClassroomHandler(classroomRepo)
	updateClassroomHandler := classroomCmd.NewUpdateClassroomHandler(classroomRepo)
	enrollStudentHandler := classroomCmd.NewEnrollStudentHandler(classroomRepo)
	listClassroomsHandler := classroomQuery.NewListTeacherClassroomsHandler(classroomRepo)
	getStumbleListHandler := classroomQuery.NewGetStumbleListHandler(classroomRepo)

	// Teacher Game Command & Query handlers
	createTeacherGameHandler := gameCmd.NewCreateTeacherGameHandler(teacherGameRepo, courseRepo)
	updateTeacherGameHandler := gameCmd.NewUpdateTeacherGameHandler(teacherGameRepo, courseRepo)
	deleteTeacherGameHandler := gameCmd.NewDeleteTeacherGameHandler(teacherGameRepo)
	listTeacherGamesHandler := gameQuery.NewListTeacherGamesHandler(teacherGameRepo)
	getTeacherGameDetailHandler := gameQuery.NewGetTeacherGameDetailHandler(teacherGameRepo)
	listAdminGamesHandler := gameQuery.NewListAdminGamesHandler(teacherGameRepo)
	listGlobalGamesHandler := gameQuery.NewListGlobalGamesHandler(teacherGameRepo)
	createGlobalGameHandler := gameCmd.NewCreateGlobalGameHandler(teacherGameRepo)
	updateGlobalGameHandler := gameCmd.NewUpdateGlobalGameHandler(teacherGameRepo)
	deleteGlobalGameHandler := gameCmd.NewDeleteGlobalGameHandler(teacherGameRepo)
	createCourseHandler := courseCmd.NewCreateCourseHandler(courseRepo, settingRepo)
	updateCourseHandler := courseCmd.NewUpdateCourseHandler(courseRepo, courseReviewRepo).WithSettingRepo(settingRepo)
	deleteCourseHandler := courseCmd.NewDeleteCourseHandler(courseRepo)
	submitCourseReviewHandler := courseCmd.NewSubmitCourseReviewHandler(courseRepo, courseReviewRepo)
	withdrawCourseReviewHandler := courseCmd.NewWithdrawCourseReviewHandler(courseReviewRepo)
	reviewCourseHandler := courseCmd.NewReviewCourseHandler(courseReviewRepo)
	listCourseReviewsHandler := courseQuery.NewListCourseReviewsHandler(courseReviewRepo)
	listTeacherCourseReviewsHandler := courseQuery.NewListTeacherCourseReviewsHandler(courseReviewRepo)
	listTeacherCoursesHandler := courseQuery.NewListTeacherCoursesHandler(courseRepo)
	listPublishedCoursesHandler := courseQuery.NewListPublishedCoursesHandler(courseRepo)
	listAdminCoursesHandler := courseQuery.NewListAdminCoursesHandler(courseRepo)
	getCourseDetailHandler := courseQuery.NewGetCourseDetailHandler(courseRepo)

	// Platform Setting Command & Query handlers
	getPublicSettingsHandler := settingQuery.NewGetPublicSettingsHandler(settingRepo)
	getAllSettingsHandler := settingQuery.NewGetAllSettingsHandler(settingRepo)
	setSettingHandler := settingCmd.NewSetPlatformSettingHandler(settingRepo)

	// Teacher Application Command & Query handlers
	applyTeacherHandler := teacherAppCmd.NewApplyTeacherHandler(teacherAppRepo, userRepo)
	reviewApplicationHandler := teacherAppCmd.NewReviewApplicationHandler(teacherAppRepo)
	deleteApplicationHandler := teacherAppCmd.NewDeleteApplicationHandler(teacherAppRepo)
	getMyApplicationHandler := teacherAppQuery.NewGetMyApplicationHandler(teacherAppRepo)
	listApplicationsHandler := teacherAppQuery.NewListApplicationsHandler(teacherAppRepo)

	// Lesson Command & Query handlers
	listCourseLessonsHandler := lessonQuery.NewListCourseLessonsHandler(lessonRepo, courseRepo)
	createLessonHandler := lessonCmd.NewCreateLessonHandler(lessonRepo, courseRepo)
	updateLessonHandler := lessonCmd.NewUpdateLessonHandler(lessonRepo, courseRepo)
	deleteLessonHandler := lessonCmd.NewDeleteLessonHandler(lessonRepo, courseRepo)
	saveLessonItemHandler := lessonCmd.NewSaveLessonItemHandler(lessonRepo, courseRepo)
	deleteLessonItemHandler := lessonCmd.NewDeleteLessonItemHandler(lessonRepo, courseRepo)
	linkLessonGameHandler := lessonCmd.NewLinkLessonGameHandler(lessonRepo, courseRepo, teacherGameRepo, lessonGameRepo)
	updateLessonGameHandler := lessonCmd.NewUpdateLessonGameHandler(lessonRepo, courseRepo, lessonGameRepo)
	unlinkLessonGameHandler := lessonCmd.NewUnlinkLessonGameHandler(lessonRepo, courseRepo, lessonGameRepo)
	listLessonGamesHandler := lessonQuery.NewListLessonGamesHandler(lessonRepo, courseRepo, lessonGameRepo)

	// Vault Command & Query handlers
	listVaultTypesHandler := vaultQuery.NewListVaultTypesHandler(contentBankRepo)
	listVaultCategoriesHandler := vaultQuery.NewListVaultCategoriesHandler(contentBankRepo)
	createVaultCategoryHandler := vaultCmd.NewCreateVaultCategoryHandler(contentBankRepo)
	listVaultItemsHandler := vaultQuery.NewListVaultItemsHandler(contentBankRepo)
	createVaultItemHandler := vaultCmd.NewCreateVaultItemHandler(contentBankRepo)
	updateVaultItemHandler := vaultCmd.NewUpdateVaultItemHandler(contentBankRepo)
	deleteVaultItemHandler := vaultCmd.NewDeleteVaultItemHandler(contentBankRepo)

	// Routers
	healthRouter := health.NewHealthRouter(db)
	lessonRouter := lesson.NewLessonRouter(
		lessonRepo,
		listCourseLessonsHandler,
		createLessonHandler,
		updateLessonHandler,
		deleteLessonHandler,
		saveLessonItemHandler,
		deleteLessonItemHandler,
	)
	updateLessonGameActivityHandler := lessonCmd.NewUpdateLessonGameActivityHandler(lessonRepo, courseRepo, teacherGameRepo)
	lessonGameRouter := lesson.NewLessonGameRouter(
		listLessonGamesHandler,
		linkLessonGameHandler,
		updateLessonGameHandler,
		unlinkLessonGameHandler,
		lessonCmd.NewCreateLessonGameHandler(lessonRepo, courseRepo, createTeacherGameHandler),
		updateLessonGameActivityHandler,
	)
	authRouter := auth.NewAuthRouter(loginWithGoogleHandler, userRepo, teacherAppRepo, jwtService, devLoginEnabled)
	courseRouter := course.NewCourseRouter(
		createCourseHandler,
		updateCourseHandler,
		deleteCourseHandler,
		submitCourseReviewHandler,
		listTeacherCoursesHandler,
		listPublishedCoursesHandler,
		listAdminCoursesHandler,
		getCourseDetailHandler,
	)
	teacherAppRouter := teacher_application.NewTeacherApplicationRouter(
		applyTeacherHandler,
		reviewApplicationHandler,
		deleteApplicationHandler,
		getMyApplicationHandler,
		listApplicationsHandler,
	)
	settingRouter := setting.NewSettingRouter(
		getPublicSettingsHandler,
		getAllSettingsHandler,
		setSettingHandler,
	)
	vaultRouter := vault.NewVaultRouter(
		listVaultTypesHandler,
		listVaultCategoriesHandler,
		createVaultCategoryHandler,
		listVaultItemsHandler,
		createVaultItemHandler,
		updateVaultItemHandler,
		deleteVaultItemHandler,
	)
	classroomRouter := classroom.NewClassroomRouter(
		createClassroomHandler,
		updateClassroomHandler,
		enrollStudentHandler,
		listClassroomsHandler,
		getStumbleListHandler,
	)
	gameRouter := gamification.NewGameRouter(
		createTeacherGameHandler,
		updateTeacherGameHandler,
		deleteTeacherGameHandler,
		listTeacherGamesHandler,
		getTeacherGameDetailHandler,
		listAdminGamesHandler,
		listGlobalGamesHandler,
		createGlobalGameHandler,
		updateGlobalGameHandler,
		deleteGlobalGameHandler,
	)
	listStudentTeacherGamesHandler := gameQuery.NewListStudentTeacherGamesHandler(classroomRepo, teacherGameRepo)
	getStudentTeacherGameDetailHandler := gameQuery.NewGetStudentTeacherGameDetailHandler(classroomRepo, teacherGameRepo)
	gameStudentRouter := gamification.NewGameStudentRouter(listStudentTeacherGamesHandler, getStudentTeacherGameDetailHandler)
	statsRepo := infraRepo.NewMySQLUserLearningStatsRepository(db)
	livesRouter := gamification.NewLivesRouter(statsRepo)
	recordGameVictoryHandler := gameCmd.NewRecordGameVictoryHandler(statsRepo)
	getUserProgressHandler := gameQuery.NewGetUserProgressHandler(statsRepo)
	progressRouter := gamification.NewProgressRouter(recordGameVictoryHandler, getUserProgressHandler)
	catalogUnlockHandler := catalogQuery.NewGetCatalogUnlockStateHandler(courseRepo, teacherGameRepo, statsRepo)
	catalogUnlockRouter := catalog.NewUnlockRouter(catalogUnlockHandler)
	swaggerRouter := docs.NewSwaggerRouter()

	// Middlewares específicos
	authRequired := restMiddleware.RequireAuth(jwtService)
	catalogRoles := restMiddleware.RequireRole(entities.RoleAdmin, entities.RoleModerator, entities.RoleStudent)
	staffRoles := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin, entities.RoleModerator)
	studentOrStaff := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin, entities.RoleModerator, entities.RoleStudent)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)
	adminOrModerator := restMiddleware.RequireRole(entities.RoleAdmin, entities.RoleModerator)

	// --- Root Status Probe (Render / Uptime monitors) ---
	rootHandler := func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"app":     "WordtapAPI",
			"status":  "operational",
			"version": "v1",
		})
	}
	e.GET("/", rootHandler)
	e.HEAD("/", rootHandler)

	// --- Grupos de enrutamiento ---
	v1 := e.Group("/api/v1")

	// --- Registro modular de rutas por dominio ---
	healthRouter.RegisterRoutes(e, v1)
	swaggerRouter.RegisterRoutes(e)
	authRouter.RegisterRoutes(v1, authRequired)
	lessonRouter.RegisterRoutes(v1, authRequired, studentOrStaff, instructorOrAdmin)
	lessonGameRouter.RegisterRoutes(v1, authRequired, staffRoles, instructorOrAdmin)
	courseRouter.RegisterRoutes(v1, authRequired, catalogRoles, studentOrStaff, staffRoles, instructorOrAdmin, adminOrModerator)
	courseReviewRouter := course.NewCourseReviewRouter(submitCourseReviewHandler, withdrawCourseReviewHandler, reviewCourseHandler, listCourseReviewsHandler, listTeacherCourseReviewsHandler)
	courseReviewRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin, adminOrModerator)
	teacherAppRouter.RegisterRoutes(v1, authRequired, adminOrModerator)
	settingRouter.RegisterRoutes(v1, authRequired, adminOrModerator)
	vaultRouter.RegisterRoutes(v1, authRequired, staffRoles, adminOrModerator, instructorOrAdmin)
	classroomRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin)
	gameRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin, adminOrModerator)
	gameStudentRouter.RegisterRoutes(v1, authRequired)
	livesRouter.RegisterRoutes(v1, authRequired)
	progressRouter.RegisterRoutes(v1, authRequired)
	catalogUnlockRouter.RegisterRoutes(v1, authRequired)

	return &Server{App: e}, nil
}

func (s *Server) Start(address string) error {
	return s.App.Start(address)
}
