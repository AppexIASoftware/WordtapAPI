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

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	catalogQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/catalog/queries"
	classroomCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	classroomQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
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
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationdispatch"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	userQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	vaultCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	vaultQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/push"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/auth"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/catalog"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/course"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/device"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/docs"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/health"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/lesson"
	notificationRoute "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/notification"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/setting"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/teacher_application"
	userRoute "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/user"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vault"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vocabulary"
	vocabQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vocabulary/queries"
)

type Server struct {
	App *echo.Echo
}

func NewServer(db *gorm.DB) (*Server, error) {
	// 1. Validate required environment variables
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

	// Global middleware
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

	// Security configuration and auxiliary services
	jwtService := security.NewJWTService(jwtSecret, 15*time.Minute, 30*24*time.Hour)
	googleVerifier := security.NewHTTPGoogleVerifier(googleClientID)

	// Infrastructure repositories
	lessonRepo := infraRepo.NewMySQLLessonRepository(db)
	userRepo := infraRepo.NewMySQLUserRepository(db)
	courseRepo := infraRepo.NewMySQLCourseRepository(db)
	courseReviewRepo := infraRepo.NewMySQLCourseReviewRepository(db)
	teacherAppRepo := infraRepo.NewMySQLTeacherApplicationRepository(db)
	settingRepo := infraRepo.NewMySQLPlatformSettingRepository(db)
	contentBankRepo := infraRepo.NewMySQLContentBankRepository(db)
	classroomRepo := infraRepo.NewMySQLClassroomRepository(db)
	classroomNotifRepo := infraRepo.NewMySQLClassroomNotificationRepository(db)
	teacherGameRepo := infraRepo.NewMySQLTeacherGameRepository(db)
	lessonGameRepo := infraRepo.NewMySQLLessonGameRepository(db)
	// Test mode keeps in-memory access to avoid writing simulated purchases
	// to the database. Otherwise, the policy uses the real repository.
	simulationEnabled := courseCmd.SimulationEnabled(appEnv, os.Getenv("ENABLE_SIMULATED_COURSE_PURCHASES"))
	var purchaseReader access.PurchaseReader
	var simulatedPurchaseStore courseCmd.SimulatedPurchaseRepository
	if simulationEnabled {
		demoPurchases := infraRepo.NewDemoPurchaseRepository()
		purchaseReader = demoPurchases
		simulatedPurchaseStore = demoPurchases
	} else {
		mysqlPurchases := infraRepo.NewMySQLSimulatedPurchaseRepository(db)
		purchaseReader = mysqlPurchases
		simulatedPurchaseStore = mysqlPurchases
	}
	contentPolicy := access.NewContentPolicy(purchaseReader, simulationEnabled)

	// Use cases
	loginWithGoogleHandler := commands.NewLoginWithGoogleHandler(userRepo, googleVerifier, jwtService, teacherAppRepo)
	// Classroom Command & Query handlers
	createClassroomHandler := classroomCmd.NewCreateClassroomHandler(classroomRepo, courseRepo)
	updateClassroomHandler := classroomCmd.NewUpdateClassroomHandler(classroomRepo, courseRepo)
	enrollStudentHandler := classroomCmd.NewEnrollStudentHandler(classroomRepo)
	listClassroomsHandler := classroomQuery.NewListTeacherClassroomsHandler(classroomRepo)
	getStumbleListHandler := classroomQuery.NewGetStumbleListHandler(classroomRepo)
	listClassroomNotificationsHandler := classroomQuery.NewListClassroomNotificationsHandler(classroomNotifRepo, classroomRepo)

	// Teacher Game Command & Query handlers
	createTeacherGameHandler := gameCmd.NewCreateTeacherGameHandler(teacherGameRepo, courseRepo).WithClassrooms(classroomRepo)
	updateTeacherGameHandler := gameCmd.NewUpdateTeacherGameHandler(teacherGameRepo, courseRepo).WithClassrooms(classroomRepo)
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
	listCourseLessonsHandler := lessonQuery.NewListCourseLessonsHandler(lessonRepo, courseRepo).WithContentPolicy(contentPolicy)
	createLessonHandler := lessonCmd.NewCreateLessonHandler(lessonRepo, courseRepo)
	updateLessonHandler := lessonCmd.NewUpdateLessonHandler(lessonRepo, courseRepo)
	deleteLessonHandler := lessonCmd.NewDeleteLessonHandler(lessonRepo, courseRepo)
	saveLessonItemHandler := lessonCmd.NewSaveLessonItemHandler(lessonRepo, courseRepo)
	deleteLessonItemHandler := lessonCmd.NewDeleteLessonItemHandler(lessonRepo, courseRepo)
	linkLessonGameHandler := lessonCmd.NewLinkLessonGameHandler(lessonRepo, courseRepo, teacherGameRepo, lessonGameRepo)
	updateLessonGameHandler := lessonCmd.NewUpdateLessonGameHandler(lessonRepo, courseRepo, lessonGameRepo)
	unlinkLessonGameHandler := lessonCmd.NewUnlinkLessonGameHandler(lessonRepo, courseRepo, lessonGameRepo)
	listLessonGamesHandler := lessonQuery.NewListLessonGamesHandler(lessonRepo, courseRepo, lessonGameRepo).WithContentPolicy(contentPolicy)

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
	).WithContentPolicy(contentPolicy)
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
	).WithContentPolicy(contentPolicy)
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
	listStudentClassroomsHandler := classroomQuery.NewListStudentClassroomsHandler(classroomRepo)
	studentClassroomRouter := classroom.NewStudentClassroomRouter(listStudentClassroomsHandler, enrollStudentHandler).WithContentPolicy(contentPolicy)
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
	courseGameAccess := gameQuery.NewCourseGameAccess(lessonRepo, infraRepo.NewGameLessonLinks(db), contentPolicy)
	listStudentTeacherGamesHandler := gameQuery.NewListStudentTeacherGamesHandler(classroomRepo, teacherGameRepo).WithCourseGameAccess(courseGameAccess)
	getStudentTeacherGameDetailHandler := gameQuery.NewGetStudentTeacherGameDetailHandler(classroomRepo, teacherGameRepo).WithCourseGameAccess(courseGameAccess)
	gameStudentRouter := gamification.NewGameStudentRouter(listStudentTeacherGamesHandler, getStudentTeacherGameDetailHandler)
	statsRepo := infraRepo.NewMySQLUserLearningStatsRepository(db)
	livesRouter := gamification.NewLivesRouter(statsRepo)
	recordGameVictoryHandler := gameCmd.NewRecordGameVictoryHandler(statsRepo)
	getUserProgressHandler := gameQuery.NewGetUserProgressHandler(statsRepo)
	progressRouter := gamification.NewProgressRouter(recordGameVictoryHandler, getUserProgressHandler)
	catalogUnlockHandler := catalogQuery.NewGetCatalogUnlockStateHandler(courseRepo, teacherGameRepo, statsRepo)
	catalogUnlockRouter := catalog.NewUnlockRouter(catalogUnlockHandler)
	swaggerRouter := docs.NewSwaggerRouter()

	// Specific middleware
	authRequired := restMiddleware.RequireAuth(jwtService)
	catalogRoles := restMiddleware.RequireRole(entities.RoleAdmin, entities.RoleModerator, entities.RoleStudent)
	staffRoles := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin, entities.RoleModerator)
	studentOrStaff := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin, entities.RoleModerator, entities.RoleStudent)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)
	adminOrModerator := restMiddleware.RequireRole(entities.RoleAdmin, entities.RoleModerator)
	adminOnly := restMiddleware.RequireRole(entities.RoleAdmin)
	studentOnly := restMiddleware.RequireRole(entities.RoleStudent)

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

	// --- Route groups ---
	v1 := e.Group("/api/v1")

	// --- Modular route registration by domain ---
	healthRouter.RegisterRoutes(e, v1)
	swaggerRouter.RegisterRoutes(e)
	authRouter.RegisterRoutes(v1, authRequired)
	lessonRouter.RegisterRoutes(v1, authRequired, studentOrStaff, instructorOrAdmin)
	lessonGameRouter.RegisterRoutes(v1, authRequired, staffRoles, instructorOrAdmin)
	courseRouter.RegisterRoutes(v1, authRequired, catalogRoles, studentOrStaff, staffRoles, instructorOrAdmin, adminOrModerator)
	simulatedPurchaseHandler := courseCmd.NewSimulatePurchaseHandler(simulatedPurchaseStore, simulationEnabled)
	course.NewSimulatedPurchaseRouter(simulatedPurchaseHandler).RegisterRoutes(v1, authRequired, studentOnly)
	courseReviewRouter := course.NewCourseReviewRouter(submitCourseReviewHandler, withdrawCourseReviewHandler, reviewCourseHandler, listCourseReviewsHandler, listTeacherCourseReviewsHandler, courseQuery.NewListReviewHistoryHandler(courseReviewRepo))
	courseReviewRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin, adminOrModerator)
	teacherAppRouter.RegisterRoutes(v1, authRequired, adminOrModerator)
	staffRepo := infraRepo.NewMySQLStaffRepository(db)
	staffRouter := userRoute.NewStaffRouter(
		userQuery.NewListStaffHandler(staffRepo),
		commands.NewProvisionStaffHandler(staffRepo),
		commands.NewUpdateStaffStatusHandler(staffRepo),
		commands.NewDeleteStaffHandler(staffRepo),
	)
	staffRouter.RegisterRoutes(v1, authRequired, adminOrModerator, adminOnly)
	settingRouter.RegisterRoutes(v1, authRequired, adminOrModerator)
	vaultRouter.RegisterRoutes(v1, authRequired, staffRoles, adminOrModerator, instructorOrAdmin)
	classroomRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin)
	studentClassroomRouter.RegisterRoutes(v1, authRequired)
	gameRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin, adminOrModerator)
	gameStudentRouter.RegisterRoutes(v1, authRequired)
	livesRouter.RegisterRoutes(v1, authRequired)
	progressRouter.RegisterRoutes(v1, authRequired)
	inboxRepo := infraRepo.NewMySQLNotificationInbox(db)
	deviceRepo := infraRepo.NewMySQLDeviceRepository(db)

	var pushSender push.PushSender
	if fcmClient, err := push.NewFCMClient(); err == nil {
		pushSender = fcmClient
	} else {
		pushSender = &push.NoOpPushSender{}
	}
	dispatchDailyWordHandler := notificationdispatch.NewDispatchDailyWordHandler(
		contentBankRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	)
	broadcastAnnouncementHandler := notificationdispatch.NewBroadcastAnnouncementHandler(
		userRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	).WithClassroomNotifRepo(classroomNotifRepo)
	broadcastClassroomHandler := classroomCmd.NewBroadcastClassroomHandler(
		classroomRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	).WithClassroomNotifRepo(classroomNotifRepo)
	dispatchVaultWordHandler := vaultCmd.NewDispatchVaultWordHandler(
		contentBankRepo,
		classroomRepo,
		userRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	)

	catalogUnlockRouter.RegisterRoutes(v1, authRequired)
	notificationRoute.NewRouter(infraRepo.NewMySQLNotificationPreferences(db)).RegisterRoutes(v1, authRequired)
	notificationRoute.NewInboxRouter(inboxRepo).RegisterRoutes(v1, authRequired)
	notificationRoute.NewDispatchRouter(dispatchDailyWordHandler).RegisterRoutes(v1, authRequired)
	notificationRoute.NewAnnouncementRouter(broadcastAnnouncementHandler, deviceRepo).RegisterRoutes(v1, authRequired, adminOrModerator)
	notificationRoute.NewClassroomNotificationRouter(listClassroomNotificationsHandler).RegisterRoutes(v1, authRequired, adminOrModerator)
	classroom.NewBroadcastRouter(broadcastClassroomHandler).RegisterRoutes(v1, authRequired, instructorOrAdmin)
	classroom.NewNotificationRouter(listClassroomNotificationsHandler).RegisterRoutes(v1, authRequired, instructorOrAdmin)
	vault.NewDispatchRouter(dispatchVaultWordHandler).RegisterRoutes(v1, authRequired, instructorOrAdmin)
	vocabulary.NewVocabularyRouter(vocabQuery.NewSearchVocabularyHandler(contentBankRepo)).RegisterRoutes(v1, authRequired)
	device.NewDeviceRouter(deviceRepo).RegisterRoutes(v1, authRequired)

	return &Server{App: e}, nil
}

func (s *Server) Start(address string) error {
	return s.App.Start(address)
}
