package teacher

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type profileUserRepoStub struct {
	repositories.UserRepository
	user *entities.User
}

func (s *profileUserRepoStub) FindByID(_ context.Context, id string) (*entities.User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return s.user, nil
}

func (s *profileUserRepoStub) UpdateUser(_ context.Context, user *entities.User) error {
	s.user = user
	return nil
}

func TestUpdateTeacherProfile_Success(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")

	repo := &profileUserRepoStub{
		user: &entities.User{
			ID:    "teacher-1",
			Email: "docente@wordtap.app",
			Name:  "Prof. Carlos",
			Role:  entities.RoleInstructor,
		},
	}

	invalidatedID := ""
	router := NewProfileRouter(repo, func(userID string) {
		invalidatedID = userID
	})

	injectUser := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-1")
			return next(c)
		}
	}
	noop := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, injectUser, noop)

	payload, _ := json.Marshal(map[string]string{
		"specialty": "Fonética y Negocios",
		"bio":       "Docente certificado TESOL con 10 años de experiencia.",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/profile", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if invalidatedID != "teacher-1" {
		t.Fatalf("expected cache invalidation for teacher-1, got %q", invalidatedID)
	}
}

func TestUpdateTeacherProfile_BioTooLong(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")
	repo := &profileUserRepoStub{
		user: &entities.User{ID: "teacher-1", Name: "Prof. Carlos", Role: entities.RoleInstructor},
	}
	router := NewProfileRouter(repo, nil)

	injectUser := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-1")
			return next(c)
		}
	}
	noop := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, injectUser, noop)

	payload, _ := json.Marshal(map[string]string{
		"bio": strings.Repeat("x", 501),
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/profile", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
