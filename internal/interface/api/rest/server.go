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

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/auth"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/course"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/health"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/lesson"
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

	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if googleClientID == "" {
		return nil, fmt.Errorf("missing required environment variable: GOOGLE_CLIENT_ID")
	}

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

	// Casos de uso
	loginWithGoogleHandler := commands.NewLoginWithGoogleHandler(userRepo, googleVerifier, jwtService)
	createCourseHandler := courseCmd.NewCreateCourseHandler(courseRepo)
	updateCourseHandler := courseCmd.NewUpdateCourseHandler(courseRepo)
	submitCourseReviewHandler := courseCmd.NewSubmitCourseReviewHandler(courseRepo)
	listTeacherCoursesHandler := courseQuery.NewListTeacherCoursesHandler(courseRepo)
	listPublishedCoursesHandler := courseQuery.NewListPublishedCoursesHandler(courseRepo)
	getCourseDetailHandler := courseQuery.NewGetCourseDetailHandler(courseRepo)

	// Routers
	healthRouter := health.NewHealthRouter(db)
	lessonRouter := lesson.NewLessonRouter(lessonRepo)
	authRouter := auth.NewAuthRouter(loginWithGoogleHandler, userRepo)
	courseRouter := course.NewCourseRouter(
		createCourseHandler,
		updateCourseHandler,
		submitCourseReviewHandler,
		listTeacherCoursesHandler,
		listPublishedCoursesHandler,
		getCourseDetailHandler,
	)

	// Middlewares específicos
	authRequired := restMiddleware.RequireAuth(jwtService)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)

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
	authRouter.RegisterRoutes(v1, authRequired)
	lessonRouter.RegisterRoutes(v1)
	courseRouter.RegisterRoutes(v1, authRequired, instructorOrAdmin)

	return &Server{App: e}, nil
}

func (s *Server) Start(address string) error {
	return s.App.Start(address)
}
