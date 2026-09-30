package auth

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type AuthRouter struct {
	loginWithGoogleHandler *commands.LoginWithGoogleHandler
	userRepo               repositories.UserRepository
}

func NewAuthRouter(
	loginWithGoogleHandler *commands.LoginWithGoogleHandler,
	userRepo repositories.UserRepository,
) *AuthRouter {
	return &AuthRouter{
		loginWithGoogleHandler: loginWithGoogleHandler,
		userRepo:               userRepo,
	}
}

// RegisterRoutes registra los endpoints del dominio de autenticación y usuarios.
func (r *AuthRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	authGroup := v1.Group("/auth")
	authGroup.POST("/google", r.LoginWithGoogle)

	usersGroup := v1.Group("/users")
	usersGroup.Use(authRequired)
	usersGroup.GET("/me", r.GetMe)
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token"`
}

// LoginWithGoogle maneja POST /api/v1/auth/google
func (r *AuthRouter) LoginWithGoogle(c *echo.Context) error {
	var req GoogleLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
	}

	token := strings.TrimSpace(req.IDToken)
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "id_token is required in request body",
		})
	}

	// Extraer y sanitizar IP de origen
	ip := c.Request().Header.Get("X-Forwarded-For")
	if ip != "" {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if ip == "" {
		ip = c.Request().RemoteAddr
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	ua := c.Request().UserAgent()

	ctx := c.Request().Context()
	result, err := r.loginWithGoogleHandler.Handle(ctx, commands.LoginWithGoogleCommand{
		IDToken:   token,
		IPAddress: ip,
		UserAgent: ua,
	})
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error":   "authentication failed",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// GetMe maneja GET /api/v1/users/me (requiere autenticación Bearer JWT)
func (r *AuthRouter) GetMe(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error": "unauthorized context",
		})
	}

	ctx := c.Request().Context()
	user, err := r.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]any{
				"error": "user not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "failed to retrieve user profile",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user": user,
	})
}
