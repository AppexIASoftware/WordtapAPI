package teacher

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type calibrationRepoStub struct {
	items map[string]*entities.TeacherReviewCalibration
}

func (s *calibrationRepoStub) GetByUserID(_ context.Context, userID string) (*entities.TeacherReviewCalibration, error) {
	return s.items[userID], nil
}

func (s *calibrationRepoStub) Upsert(_ context.Context, c *entities.TeacherReviewCalibration) error {
	if s.items == nil {
		s.items = make(map[string]*entities.TeacherReviewCalibration)
	}
	s.items[c.UserID] = c
	return nil
}

func TestCalibrationRouter_GetDefaultsAndSave(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")
	repo := &calibrationRepoStub{}
	router := NewCalibrationRouter(repo)

	injectUser := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-1")
			return next(c)
		}
	}
	noop := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, injectUser, noop)

	// 1. Sin registro previo devuelve valores por defecto
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/review-calibration", nil)
	recGet := httptest.NewRecorder()
	e.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recGet.Code)
	}
	var got ReviewCalibrationPayload
	if err := json.Unmarshal(recGet.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.EaseFactor != defaultEaseFactor || got.Quality != defaultQuality || got.Repetitions != defaultRepetitions {
		t.Fatalf("unexpected defaults: %+v", got)
	}

	// 2. Guardar calibración válida persiste y se recupera luego
	body, _ := json.Marshal(ReviewCalibrationPayload{EaseFactor: 2.2, Quality: 5, Repetitions: 4})
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/review-calibration", bytes.NewReader(body))
	reqPut.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recPut := httptest.NewRecorder()
	e.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recPut.Code, recPut.Body.String())
	}

	reqGet2 := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/review-calibration", nil)
	recGet2 := httptest.NewRecorder()
	e.ServeHTTP(recGet2, reqGet2)
	if err := json.Unmarshal(recGet2.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.EaseFactor != 2.2 || got.Quality != 5 || got.Repetitions != 4 {
		t.Fatalf("expected persisted calibration, got %+v", got)
	}
}

func TestCalibrationRouter_RejectsOutOfRange(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")
	router := NewCalibrationRouter(&calibrationRepoStub{})

	injectUser := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-1")
			return next(c)
		}
	}
	noop := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, injectUser, noop)

	body, _ := json.Marshal(ReviewCalibrationPayload{EaseFactor: 1.0, Quality: 4, Repetitions: 3})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/review-calibration", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
