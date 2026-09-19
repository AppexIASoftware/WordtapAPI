package rest

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/health"
)

type Server struct {
	App *echo.Echo
}

func NewServer(db *gorm.DB) *Server {
	e := echo.New()

	// Middlewares
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS("*"))

	// Grupo base de rutas de la API
	api := e.Group("/api/services/v1")

	// Registro de rutas
	healthRouter := health.NewHealthRouter(db)
	api.GET("/health", healthRouter.CheckHealth)

	return &Server{App: e}
}

func (s *Server) Start(address string) error {
	return s.App.Start(address)
}
