package auth

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type userCacheEntry struct {
	user      *entities.User
	expiresAt time.Time
}

type AuthRouter struct {
	loginWithGoogleHandler *commands.LoginWithGoogleHandler
	userRepo               repositories.UserRepository
	teacherAppRepo         repositories.TeacherApplicationRepository
	jwtService             *security.JWTService
	devLoginEnabled        bool
	userCache              sync.Map
}

func NewAuthRouter(
	loginWithGoogleHandler *commands.LoginWithGoogleHandler,
	userRepo repositories.UserRepository,
	teacherAppRepo repositories.TeacherApplicationRepository,
	jwtService *security.JWTService,
	devLoginEnabled bool,
) *AuthRouter {
	return &AuthRouter{
		loginWithGoogleHandler: loginWithGoogleHandler,
		userRepo:               userRepo,
		teacherAppRepo:         teacherAppRepo,
		jwtService:             jwtService,
		devLoginEnabled:        devLoginEnabled,
	}
}

// RegisterRoutes registra los endpoints del dominio de autenticación y usuarios.
func (r *AuthRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	authGroup := v1.Group("/auth")
	authGroup.POST("/google", r.LoginWithGoogle)
	authGroup.POST("/refresh", r.Refresh)
	authGroup.POST("/logout", r.Logout)
	if r.devLoginEnabled {
		authGroup.POST("/dev-login", r.DevLogin)
	}

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
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	token := strings.TrimSpace(req.IDToken)
	if token == "" {
		return response.Fail(c, http.StatusBadRequest, "No se recibió la credencial de acceso.")
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
		if errors.Is(err, commands.ErrGoogleEmailNotVerified) {
			return response.Fail(c, http.StatusForbidden, "Tu correo de Google no está verificado.")
		}
		if errors.Is(err, commands.ErrAccountLinkRequired) {
			return response.Fail(c, http.StatusConflict, "Ya existe una cuenta con ese correo. Iniciá sesión con tu método original o contactá a soporte para vincularla.")
		}
		if strings.HasPrefix(err.Error(), "account_suspended:") {
			reason := strings.TrimSpace(strings.TrimPrefix(err.Error(), "account_suspended:"))
			return c.JSON(http.StatusForbidden, map[string]any{
				"error":   "account_suspended",
				"message": fmt.Sprintf("Tu cuenta docente está suspendida. Motivo: %s", reason),
				"reason":  reason,
			})
		}
		return response.Error(c, http.StatusUnauthorized, "No se pudo iniciar sesión.", err)
	}

	return c.JSON(http.StatusOK, result)
}

// GetMe maneja GET /api/v1/users/me (requiere autenticación Bearer JWT)
func (r *AuthRouter) GetMe(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	ctx := c.Request().Context()
	var user *entities.User
	if val, ok := r.userCache.Load(userID); ok {
		entry := val.(userCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			user = entry.user
		}
	}

	if user == nil {
		var err error
		user, err = r.userRepo.FindByID(ctx, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.Fail(c, http.StatusNotFound, "No se encontró el usuario.")
			}
			return response.Error(c, http.StatusInternalServerError, "No se pudo cargar tu perfil.", err)
		}

		if user.IsActive {
			r.userCache.Store(userID, userCacheEntry{
				user:      user,
				expiresAt: time.Now().Add(30 * time.Second),
			})
		}
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
		return c.JSON(http.StatusOK, map[string]any{
			"user": user,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user":         user,
		"access_token": accessToken,
		"expires_in":   expiresIn,
	})
}

type DevLoginRequest struct {
	Role  string `json:"role"`
	Email string `json:"email"`
}

// DevLogin maneja POST /api/v1/auth/dev-login para emitir credenciales válidas en entornos de desarrollo/sandbox.
func (r *AuthRouter) DevLogin(c *echo.Context) error {
	if !r.devLoginEnabled {
		return response.Fail(c, http.StatusNotFound, "Recurso no encontrado.")
	}

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
			return response.Error(c, http.StatusInternalServerError, "No se pudo crear el usuario de desarrollo.", err)
		}
	} else if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo consultar el usuario.", err)
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
		return response.Error(c, http.StatusInternalServerError, "No se pudo iniciar la sesión.", err)
	}

	refreshToken, refreshHash, expiresAt, err := r.jwtService.GenerateRefreshToken()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo iniciar la sesión.", err)
	}

	session := &entities.UserSession{
		UserID:           user.ID,
		SessionTokenHash: refreshHash,
		ExpiresAt:        expiresAt,
	}
	if err := r.userRepo.CreateSession(ctx, session); err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo iniciar la sesión.", err)
	}

	return c.JSON(http.StatusOK, commands.AuthResultDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         commands.ToAuthUserDTO(user),
	})
}
