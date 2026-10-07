package health

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

type HealthRouter struct {
	db *gorm.DB
}

func NewHealthRouter(db *gorm.DB) *HealthRouter {
	return &HealthRouter{db: db}
}

// RegisterRoutes registra los endpoints de monitoreo y diagnóstico de infraestructura.
func (h *HealthRouter) RegisterRoutes(e *echo.Echo, v1 *echo.Group) {
	e.GET("/health", h.CheckHealth)
	e.GET("/healthz", h.CheckHealth)

	if v1 != nil {
		v1.GET("/health", h.CheckHealth)
	}
}

// En Echo v5 se recibe *echo.Context (puntero a struct)
func (h *HealthRouter) CheckHealth(c *echo.Context) error {
	start := time.Now()

	dbConnected := false
	var pingError string

	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			pingError = err.Error()
		} else if err := sqlDB.Ping(); err != nil {
			// Si una conexión inactiva expiró en el servidor remoto, reintentar con una conexión fresca del pool
			if retryErr := sqlDB.Ping(); retryErr != nil {
				pingError = retryErr.Error()
			} else {
				dbConnected = true
			}
		} else {
			dbConnected = true
		}
	} else {
		pingError = "database connection is uninitialized"
	}

	elapsed := time.Since(start).Milliseconds()

	status := http.StatusOK
	if !dbConnected {
		status = http.StatusServiceUnavailable
	}

	// En Echo v5 se usa map[string]any nativo de Go en lugar de echo.Map
	return c.JSON(status, map[string]any{
		"status":    "ok",
		"message":   "Wordtap API is operational",
		"timestamp": time.Now().Format(time.RFC3339),
		"database": map[string]any{
			"connected": dbConnected,
			"engine":    "mysql",
			"ping_ms":   elapsed,
			"error":     pingError,
		},
	})
}
