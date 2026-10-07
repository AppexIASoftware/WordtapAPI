package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
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
			c.Set("user_role", string(claims.Role))
			c.Set("access_tier", string(claims.AccessTier))

			return next(c)
		}
	}
}

// OptionalAuth decodifica el token Bearer JWT si existe en Authorization sin bloquear peticiones anónimas.
func OptionalAuth(jwtService *security.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return next(c)
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return next(c)
			}

			tokenString := strings.TrimSpace(parts[1])
			claims, err := jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				return next(c)
			}

			c.Set("user_id", claims.Subject)
			c.Set("user_email", claims.Email)
			c.Set("user_role", string(claims.Role))
			c.Set("access_tier", string(claims.AccessTier))

			return next(c)
		}
	}
}

// RequireRole restringe el acceso solo a usuarios con los roles autorizados.
func RequireRole(allowedRoles ...entities.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			roleVal := c.Get("user_role")
			roleStr, _ := roleVal.(string)

			for _, allowed := range allowedRoles {
				if strings.EqualFold(roleStr, string(allowed)) {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]any{
				"error": "insufficient permissions for this resource",
			})
		}
	}
}
