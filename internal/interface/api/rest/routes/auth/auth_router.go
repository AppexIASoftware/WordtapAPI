package auth

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
)

type AuthRouter struct {
	loginWithGoogleHandler *commands.LoginWithGoogleHandler
	userRepo               repositories.UserRepository
	teacherAppRepo         repositories.TeacherApplicationRepository
	jwtService             *security.JWTService
}

func NewAuthRouter(
	loginWithGoogleHandler *commands.LoginWithGoogleHandler,
	userRepo repositories.UserRepository,
	teacherAppRepo repositories.TeacherApplicationRepository,
	jwtService *security.JWTService,
) *AuthRouter {
	return &AuthRouter{
		loginWithGoogleHandler: loginWithGoogleHandler,
		userRepo:               userRepo,
		teacherAppRepo:         teacherAppRepo,
		jwtService:             jwtService,
	}
}

// RegisterRoutes registra los endpoints del dominio de autenticación y usuarios.
func (r *AuthRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	authGroup := v1.Group("/auth")
	authGroup.POST("/google", r.LoginWithGoogle)
	authGroup.POST("/dev-login", r.DevLogin)

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
		if strings.HasPrefix(err.Error(), "account_suspended:") {
			reason := strings.TrimSpace(strings.TrimPrefix(err.Error(), "account_suspended:"))
			return c.JSON(http.StatusForbidden, map[string]any{
				"error":   "account_suspended",
				"message": fmt.Sprintf("Tu cuenta docente está suspendida. Motivo: %s", reason),
				"reason":  reason,
			})
		}
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

	if !user.IsActive {
		reason := "Suspensión administrativa preventiva"
		if r.teacherAppRepo != nil {
			if app, err := r.teacherAppRepo.FindByUserID(ctx, user.ID); err == nil && app != nil && app.RejectionReason != nil && *app.RejectionReason != "" {
				reason = *app.RejectionReason
			}
		}
		return c.JSON(http.StatusForbidden, map[string]any{
			"error":   "account_suspended",
			"message": fmt.Sprintf("Tu cuenta docente está suspendida. Motivo: %s", reason),
			"reason":  reason,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user": user,
	})
}

type DevLoginRequest struct {
	Role  string `json:"role"`
	Email string `json:"email"`
}

// DevLogin maneja POST /api/v1/auth/dev-login para emitir credenciales válidas en entornos de desarrollo/sandbox.
func (r *AuthRouter) DevLogin(c *echo.Context) error {
	var req DevLoginRequest
	_ = c.Bind(&req)

	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "instructor"
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		switch role {
		case "admin":
			email = "admin@wordtap.app"
		case "moderator":
			email = "elena.ramos@wordtap.app"
		default:
			role = "instructor"
			email = "mateo.silva@wordtap.app"
		}
	}

	ctx := c.Request().Context()
	user, err := r.userRepo.FindByEmail(ctx, email)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now()
		targetRole := entities.RoleInstructor
		targetTier := entities.AccessTierCourse
		switch role {
		case "admin":
			targetRole = entities.RoleAdmin
			targetTier = entities.AccessTierAdmin
		case "moderator":
			targetRole = entities.RoleModerator
			targetTier = entities.AccessTierCourse
		}

		name := strings.Split(email, "@")[0]
		user = &entities.User{
			Email:             email,
			Name:              name,
			Role:              targetRole,
			AccessTier:        targetTier,
			PreferredLanguage: "es",
			LearningLevel:     "advanced",
			Timezone:          "UTC",
			IsActive:          true,
			EmailVerifiedAt:   &now,
		}
		authAccount := &entities.UserAuthAccount{
			Provider:          "dev",
			ProviderAccountID: email,
		}
		if err := r.userRepo.CreateWithInitialState(ctx, user, authAccount); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "failed to create dev user",
			})
		}
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "database error finding user",
		})
	}

	if !user.IsActive {
		reason := "Suspensión administrativa preventiva"
		if r.teacherAppRepo != nil {
			if app, err := r.teacherAppRepo.FindByUserID(ctx, user.ID); err == nil && app != nil && app.RejectionReason != nil && *app.RejectionReason != "" {
				reason = *app.RejectionReason
			}
		}
		return c.JSON(http.StatusForbidden, map[string]any{
			"error":   "account_suspended",
			"message": fmt.Sprintf("Tu cuenta docente está suspendida. Motivo: %s", reason),
			"reason":  reason,
		})
	}

	accessToken, expiresIn, err := r.jwtService.GenerateAccessToken(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to generate access token"})
	}

	refreshToken, refreshHash, expiresAt, err := r.jwtService.GenerateRefreshToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to generate refresh token"})
	}

	session := &entities.UserSession{
		UserID:           user.ID,
		SessionTokenHash: refreshHash,
		ExpiresAt:        expiresAt,
	}
	_ = r.userRepo.CreateSession(ctx, session)

	return c.JSON(http.StatusOK, commands.AuthResultDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User: commands.AuthUserDTO{
			ID:                user.ID,
			Email:             user.Email,
			Name:              user.Name,
			AvatarURL:         user.AvatarURL,
			Role:              user.Role,
			AccessTier:        user.AccessTier,
			PreferredLanguage: user.PreferredLanguage,
			LearningLevel:     user.LearningLevel,
			Timezone:          user.Timezone,
		},
	})
}
