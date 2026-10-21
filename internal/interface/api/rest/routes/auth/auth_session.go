package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh maneja POST /api/v1/auth/refresh. Rota el refresh token de forma
// atómica: revoca el presentado y persiste el nuevo en una sola transacción.
func (r *AuthRouter) Refresh(c *echo.Context) error {
	var req RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		return response.Fail(c, http.StatusBadRequest, "No se recibió la sesión a renovar.")
	}

	ctx := c.Request().Context()
	tokenHash := security.HashToken(token)

	session, err := r.userRepo.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, http.StatusUnauthorized, "Tu sesión expiró. Iniciá sesión de nuevo.")
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo renovar la sesión.", err)
	}
	if session.User == nil {
		return response.Fail(c, http.StatusUnauthorized, "Tu sesión expiró. Iniciá sesión de nuevo.")
	}
	if !session.User.IsActive {
		reason := "Suspensión administrativa preventiva"
		if r.teacherAppRepo != nil {
			if app, err := r.teacherAppRepo.FindByUserID(ctx, session.UserID); err == nil && app != nil && app.RejectionReason != nil && *app.RejectionReason != "" {
				reason = *app.RejectionReason
			}
		}
		return c.JSON(http.StatusForbidden, map[string]any{
			"error":   "account_suspended",
			"message": fmt.Sprintf("Tu cuenta docente está suspendida. Motivo: %s", reason),
			"reason":  reason,
		})
	}

	// Generar ambos tokens antes de rotar evita revocar la sesión si la emisión falla.
	accessToken, expiresIn, err := r.jwtService.GenerateAccessToken(session.User)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo renovar la sesión.", err)
	}

	refreshToken, refreshHash, expiresAt, err := r.jwtService.GenerateRefreshToken()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo renovar la sesión.", err)
	}

	newSession := &entities.UserSession{
		UserID:           session.UserID,
		SessionTokenHash: refreshHash,
		ExpiresAt:        expiresAt,
		IPAddress:        session.IPAddress,
		UserAgent:        session.UserAgent,
	}
	rotated, err := r.userRepo.RotateSession(ctx, tokenHash, newSession)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo renovar la sesión.", err)
	}
	if !rotated {
		return response.Fail(c, http.StatusUnauthorized, "Tu sesión expiró. Iniciá sesión de nuevo.")
	}

	return c.JSON(http.StatusOK, commands.AuthResultDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         commands.ToAuthUserDTO(session.User),
	})
}

// Logout maneja POST /api/v1/auth/logout. Revoca el refresh token presentado.
// Es idempotente: un token ya revocado o inexistente también devuelve 204.
func (r *AuthRouter) Logout(c *echo.Context) error {
	var req RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		return response.Fail(c, http.StatusBadRequest, "No se recibió la sesión a cerrar.")
	}

	if err := r.userRepo.RevokeSession(c.Request().Context(), security.HashToken(token)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cerrar la sesión.", err)
	}

	return c.NoContent(http.StatusNoContent)
}
