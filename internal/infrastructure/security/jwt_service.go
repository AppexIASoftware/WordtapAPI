package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrInvalidClaims = errors.New("invalid token claims")
)

type UserClaims struct {
	Email      string              `json:"email"`
	AccessTier entities.AccessTier `json:"access_tier"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

func NewJWTService(secret string, accessTTL, refreshTTL time.Duration) *JWTService {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	return &JWTService{
		secretKey:       []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		issuer:          "wordtap-api",
	}
}

// GenerateAccessToken emite un JWT firmado para el usuario con claims de perfil y membresía.
func (s *JWTService) GenerateAccessToken(user *entities.User) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessTokenTTL)

	claims := UserClaims{
		Email:      user.Email,
		AccessTier: user.AccessTier,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, int64(s.accessTokenTTL.Seconds()), nil
}

// GenerateRefreshToken genera un token opaco criptográficamente aleatorio y su hash SHA-256.
func (s *JWTService) GenerateRefreshToken() (token string, hash string, expiresAt time.Time, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	token = hex.EncodeToString(bytes)
	hash = HashToken(token)
	expiresAt = time.Now().Add(s.refreshTokenTTL)
	return token, hash, expiresAt, nil
}

// ValidateAccessToken verifica la firma y vigencia del JWT, retornando los claims del usuario.
func (s *JWTService) ValidateAccessToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// HashToken calcula el hash SHA-256 de un token para almacenamiento seguro en base de datos.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
