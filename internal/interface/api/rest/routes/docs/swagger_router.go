package docs

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

type SwaggerRouter struct{}

func NewSwaggerRouter() *SwaggerRouter {
	return &SwaggerRouter{}
}

// RegisterRoutes registra la interfaz gráfica Swagger UI y la especificación OpenAPI.
func (r *SwaggerRouter) RegisterRoutes(e *echo.Echo) {
	e.GET("/swagger", r.SwaggerUI)
	e.GET("/swagger/", r.SwaggerUI)
	e.GET("/swagger/index.html", r.SwaggerUI)
	e.GET("/swagger/doc.json", r.SwaggerSpec)
}

func (r *SwaggerRouter) SwaggerUI(c *echo.Context) error {
	return c.HTML(http.StatusOK, SwaggerHTML)
}

func (r *SwaggerRouter) SwaggerSpec(c *echo.Context) error {
	return c.Blob(http.StatusOK, "application/json; charset=utf-8", []byte(SwaggerJSON))
}
