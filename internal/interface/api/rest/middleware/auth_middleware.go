package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
)

// RequireAuth valida el token Bearer JWT en el encabezado Authorization e inyecta la identidad en el contexto.
func RequireAuth(jwtService *security.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"error": "missing authorization header",
				})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"error": "invalid authorization format, expected 'Bearer <token>'",
				})
			}

			tokenString := strings.TrimSpace(parts[1])
			claims, err := jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"error": "invalid or expired token",
				})
			}

			// Inyectar datos del usuario autenticado en el contexto de la petición
			c.Set("user_id", claims.Subject)
			c.Set("user_email", claims.Email)
			c.Set("access_tier", string(claims.AccessTier))

			return next(c)
		}
	}
}
