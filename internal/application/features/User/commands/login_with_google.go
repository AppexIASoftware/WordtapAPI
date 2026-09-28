package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
)

type LoginWithGoogleCommand struct {
	IDToken   string
	IPAddress string
	UserAgent string
}

type AuthUserDTO struct {
	ID                string              `json:"id"`
	Email             string              `json:"email"`
	Name              string              `json:"name"`
	AvatarURL         *string             `json:"avatar_url"`
	AccessTier        entities.AccessTier `json:"access_tier"`
	PreferredLanguage string              `json:"preferred_language"`
	LearningLevel     string              `json:"learning_level"`
	Timezone          string              `json:"timezone"`
}

type AuthResultDTO struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
	TokenType    string      `json:"token_type"`
	User         AuthUserDTO `json:"user"`
}

type LoginWithGoogleHandler struct {
	userRepo       repositories.UserRepository
	googleVerifier security.GoogleVerifier
	jwtService     *security.JWTService
}

func NewLoginWithGoogleHandler(
	userRepo repositories.UserRepository,
	googleVerifier security.GoogleVerifier,
	jwtService *security.JWTService,
) *LoginWithGoogleHandler {
	return &LoginWithGoogleHandler{
		userRepo:       userRepo,
		googleVerifier: googleVerifier,
		jwtService:     jwtService,
	}
}

func (h *LoginWithGoogleHandler) Handle(ctx context.Context, cmd LoginWithGoogleCommand) (*AuthResultDTO, error) {
	claims, err := h.googleVerifier.VerifyIDToken(ctx, cmd.IDToken)
	if err != nil {
		return nil, fmt.Errorf("google verification failed: %w", err)
	}

	var user *entities.User

	// 1. Buscar si ya existe la cuenta OAuth de Google vinculada
	account, err := h.userRepo.FindAuthAccount(ctx, "google", claims.Sub)
	if err == nil && account != nil {
		if account.User == nil {
			return nil, errors.New("associated user record not found for auth account")
		}
		user = account.User
		// Actualizar avatar si no tenía
		if user.AvatarURL == nil && claims.Picture != "" {
			user.AvatarURL = &claims.Picture
			_ = h.userRepo.UpdateUser(ctx, user)
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) || account == nil {
		// 2. Verificar si existe usuario por email
		existingUser, errEmail := h.userRepo.FindByEmail(ctx, claims.Email)
		if errEmail == nil && existingUser != nil {
			user = existingUser
			// Vincular cuenta Google al usuario existente
			authAccount := &entities.UserAuthAccount{
				UserID:            user.ID,
				Provider:          "google",
				ProviderAccountID: claims.Sub,
			}
			if err := h.userRepo.CreateAuthAccount(ctx, authAccount); err != nil {
				return nil, fmt.Errorf("failed to link google account: %w", err)
			}
		} else {
			// 3. Crear nuevo usuario con estado inicial completo
			userName := claims.Name
			if userName == "" {
				userName = strings.Split(claims.Email, "@")[0]
			}

			var avatar *string
			if claims.Picture != "" {
				avatar = &claims.Picture
			}

			now := time.Now()
			newUser := &entities.User{
				Email:             claims.Email,
				Name:              userName,
				AvatarURL:         avatar,
				AccessTier:        entities.AccessTierFree,
				PreferredLanguage: "es",
				LearningLevel:     "beginner",
				Timezone:          "UTC",
				IsActive:          true,
				EmailVerifiedAt:   &now,
			}

			authAccount := &entities.UserAuthAccount{
				Provider:          "google",
				ProviderAccountID: claims.Sub,
			}

			if err := h.userRepo.CreateWithInitialState(ctx, newUser, authAccount); err != nil {
				return nil, fmt.Errorf("failed to register new user: %w", err)
			}
			user = newUser
		}
	} else {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	// 4. Generar tokens JWT y Refresh
	accessToken, expiresIn, err := h.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshHash, expiresAt, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 5. Registrar sesión persistente
	var ip *string
	if cmd.IPAddress != "" {
		ip = &cmd.IPAddress
	}
	var ua *string
	if cmd.UserAgent != "" {
		truncatedUA := cmd.UserAgent
		if len(truncatedUA) > 500 {
			truncatedUA = truncatedUA[:500]
		}
		ua = &truncatedUA
	}

	session := &entities.UserSession{
		UserID:           user.ID,
		SessionTokenHash: refreshHash,
		ExpiresAt:        expiresAt,
		IPAddress:        ip,
		UserAgent:        ua,
	}
	if err := h.userRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to persist user session: %w", err)
	}

	return &AuthResultDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User: AuthUserDTO{
			ID:                user.ID,
			Email:             user.Email,
			Name:              user.Name,
			AvatarURL:         user.AvatarURL,
			AccessTier:        user.AccessTier,
			PreferredLanguage: user.PreferredLanguage,
			LearningLevel:     user.LearningLevel,
			Timezone:          user.Timezone,
		},
	}, nil
}
