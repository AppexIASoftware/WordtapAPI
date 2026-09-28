package rest

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/User/commands"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/auth"
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

	// Casos de uso
	loginWithGoogleHandler := commands.NewLoginWithGoogleHandler(userRepo, googleVerifier, jwtService)

	// Routers
	healthRouter := health.NewHealthRouter(db)
	lessonRouter := lesson.NewLessonRouter(lessonRepo)
	authRouter := auth.NewAuthRouter(loginWithGoogleHandler, userRepo)

	// Middlewares específicos
	authRequired := restMiddleware.RequireAuth(jwtService)

	// --- Rutas de diagnóstico ---
	servicesV1 := e.Group("/api/services/v1")
	servicesV1.GET("/health", healthRouter.CheckHealth)
	servicesV1.GET("/lessons/:id", lessonRouter.GetLessonDetail)

	// --- API REST Canónica v1 ---
	v1 := e.Group("/api/v1")

	// Autenticación pública
	authGroup := v1.Group("/auth")
	authGroup.POST("/google", authRouter.LoginWithGoogle)

	// Usuarios y perfiles protegidos
	usersGroup := v1.Group("/users")
	usersGroup.Use(authRequired)
	usersGroup.GET("/me", authRouter.GetMe)

	// Catálogo y lecciones
	lessonsGroup := v1.Group("/lessons")
	lessonsGroup.GET("/:id", lessonRouter.GetLessonDetail)

	return &Server{App: e}, nil
}

func (s *Server) Start(address string) error {
	return s.App.Start(address)
}
