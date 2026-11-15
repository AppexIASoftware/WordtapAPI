package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// ActiveUserChecker verifies whether an authenticated user ID is still active.
type ActiveUserChecker func(ctx context.Context, userID string) (bool, error)

// RequireAuth valida el token Bearer JWT en el encabezado Authorization e inyecta la identidad en el contexto.
func RequireAuth(jwtService *security.JWTService, checkers ...ActiveUserChecker) echo.MiddlewareFunc {
	var checkActive ActiveUserChecker
	if len(checkers) > 0 {
		checkActive = checkers[0]
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
			}

			tokenString := strings.TrimSpace(parts[1])
			claims, err := jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				return response.Fail(c, http.StatusUnauthorized, "Tu sesión expiró. Inicia sesión nuevamente.")
			}

			if checkActive != nil {
				active, err := checkActive(c.Request().Context(), claims.Subject)
				if err != nil {
					return response.Error(c, http.StatusInternalServerError, "No se pudo validar el estado de tu cuenta.", err)
				}
				if !active {
					return c.JSON(http.StatusForbidden, map[string]any{
						"error":   "account_suspended",
						"message": "Tu cuenta está suspendida.",
					})
				}
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

			return response.Fail(c, http.StatusForbidden, "No tienes permisos para realizar esta acción.")
		}
	}
}

// RequirePermission restringe el acceso evaluando dinámicamente la política RBAC para el rol autenticado.
func RequirePermission(evaluator *access.RBACPolicyEvaluator, perm access.RBACPermission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			roleVal := c.Get("user_role")
			roleStr, _ := roleVal.(string)
			role := entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr)))

			if evaluator != nil && evaluator.IsAllowed(c.Request().Context(), role, perm) {
				return next(c)
			}

			return response.Fail(c, http.StatusForbidden, "No tienes permisos para realizar esta acción.")
		}
	}
}
