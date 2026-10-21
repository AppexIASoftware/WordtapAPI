package gamification_test

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

func bearerRequest(method, target string, body []byte, token string) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestAdminGameCRUD_GlobalGames(t *testing.T) {
	e, jwtSvc, repo := setupAdminGameRouter()

	adminToken, _, err := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "admin-root",
		Email: "admin@wordtap.app",
		Role:  entities.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	t.Run("POST /api/v1/admin/games creates a platform-global activity", func(t *testing.T) {
		payload := map[string]any{
			"title":              "Plataforma: Verbos Irregulares",
			"base_archetype":     "match",
			"time_limit_seconds": 30,
			"reward_xp":          25,
		}
		body, _ := json.Marshal(payload)

		req := bearerRequest(http.MethodPost, "/api/v1/admin/games", body, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var payloadRes struct {
			Success bool           `json:"success"`
			Data    map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payloadRes); err != nil {
			t.Fatalf("failed to decode created activity: %v", err)
		}
		if !payloadRes.Success {
			t.Fatalf("expected success envelope, got %s", rec.Body.String())
		}
		if payloadRes.Data["is_global"] != true {
			t.Errorf("expected is_global true, got %v", payloadRes.Data["is_global"])
		}
		if payloadRes.Data["course_id"] != nil {
			t.Errorf("expected course_id null for a global activity, got %v", payloadRes.Data["course_id"])
		}
		if payloadRes.Data["teacher_id"] != "admin-root" {
			t.Errorf("expected author admin-root, got %v", payloadRes.Data["teacher_id"])
		}
	})

	t.Run("PUT /api/v1/admin/games/:id updates a global activity", func(t *testing.T) {
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:        "game-global-update",
			TeacherID: "admin-root",
			Title:     "Global Original",
			OwnerType: entities.OwnerTypePlatform,
			IsGlobal:  true,
		})

		body, _ := json.Marshal(map[string]any{
			"title":              "Global Actualizada",
			"time_limit_seconds": 60,
		})

		req := bearerRequest(http.MethodPut, "/api/v1/admin/games/game-global-update", body, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		stored, _ := repo.FindByID(context.Background(), "game-global-update")
		if stored == nil || stored.Title != "Global Actualizada" {
			t.Fatalf("expected updated title, got %+v", stored)
		}
		if stored.TimeLimitSeconds != 60 {
			t.Errorf("expected time limit 60, got %d", stored.TimeLimitSeconds)
		}
	})

	t.Run("PUT rejects invalid custom_words_json without overwriting the stored words", func(t *testing.T) {
		words := `[{"term":"live","translation":"vivir"}]`
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:              "game-global-json",
			TeacherID:       "admin-root",
			Title:           "Global JSON",
			AllowedMistakes: 3,
			CustomWordsJSON: &words,
			OwnerType:       entities.OwnerTypePlatform,
			IsGlobal:        true,
		})

		body, _ := json.Marshal(map[string]any{"custom_words_json": "{no-es-json"})
		req := bearerRequest(http.MethodPut, "/api/v1/admin/games/game-global-json", body, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}

		stored, _ := repo.FindByID(context.Background(), "game-global-json")
		if stored == nil || stored.CustomWordsJSON == nil || *stored.CustomWordsJSON != words {
			t.Fatalf("stored words must survive an invalid payload, got %+v", stored)
		}
	})

	t.Run("PUT keeps allowed_mistakes when omitted and accepts an explicit zero", func(t *testing.T) {
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:              "game-global-mistakes",
			TeacherID:       "admin-root",
			Title:           "Global Mistakes",
			AllowedMistakes: 2,
			OwnerType:       entities.OwnerTypePlatform,
			IsGlobal:        true,
		})

		omitted, _ := json.Marshal(map[string]any{"title": "Solo título"})
		req := bearerRequest(http.MethodPut, "/api/v1/admin/games/game-global-mistakes", omitted, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored, _ := repo.FindByID(context.Background(), "game-global-mistakes")
		if stored == nil || stored.AllowedMistakes != 2 {
			t.Fatalf("omitted field must keep 2, got %+v", stored)
		}

		explicitZero, _ := json.Marshal(map[string]any{"allowed_mistakes": 0})
		req = bearerRequest(http.MethodPut, "/api/v1/admin/games/game-global-mistakes", explicitZero, adminToken)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored, _ = repo.FindByID(context.Background(), "game-global-mistakes")
		if stored == nil || stored.AllowedMistakes != 0 {
			t.Fatalf("explicit zero must be stored, got %+v", stored)
		}
	})

	t.Run("PUT /api/v1/admin/games/:id refuses a teacher-owned activity", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"title": "Intento"})

		req := bearerRequest(http.MethodPut, "/api/v1/admin/games/game-admin-1", body, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}

		var errRes struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &errRes); err != nil {
			t.Fatalf("failed to decode error envelope: %v", err)
		}
		if errRes.Error != "Solo se pueden gestionar actividades de la plataforma." {
			t.Errorf("unexpected error message: %q", errRes.Error)
		}

		stored, _ := repo.FindByID(context.Background(), "game-admin-1")
		if stored == nil || stored.Title == "Intento" {
			t.Fatalf("teacher-owned activity must not be modified, got %+v", stored)
		}
	})

	t.Run("DELETE /api/v1/admin/games/:id deletes a global activity", func(t *testing.T) {
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:        "game-global-delete",
			TeacherID: "admin-root",
			Title:     "Global Para Borrar",
			OwnerType: entities.OwnerTypePlatform,
			IsGlobal:  true,
		})

		req := bearerRequest(http.MethodDelete, "/api/v1/admin/games/game-global-delete", nil, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		stored, _ := repo.FindByID(context.Background(), "game-global-delete")
		if stored != nil {
			t.Fatalf("expected the global activity to be deleted, got %+v", stored)
		}
	})

	t.Run("DELETE /api/v1/admin/games/:id archives a global activity in use", func(t *testing.T) {
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:        "game-global-archived",
			TeacherID: "admin-root",
			Title:     "Global En Uso",
			OwnerType: entities.OwnerTypePlatform,
			IsGlobal:  true,
			IsActive:  true,
		})
		repo.usage["game-global-archived"] = 3

		req := bearerRequest(http.MethodDelete, "/api/v1/admin/games/game-global-archived", nil, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var payloadRes struct {
			Success bool           `json:"success"`
			Message string         `json:"message"`
			Data    map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payloadRes); err != nil {
			t.Fatalf("failed to decode archive envelope: %v", err)
		}
		if payloadRes.Data["archived"] != true {
			t.Errorf("expected archived true, got %v", payloadRes.Data["archived"])
		}
		if payloadRes.Data["usage_count"] != float64(3) {
			t.Errorf("expected usage_count 3, got %v", payloadRes.Data["usage_count"])
		}

		stored, _ := repo.FindByID(context.Background(), "game-global-archived")
		if stored == nil {
			t.Fatal("an activity in use must be kept")
		}
		if stored.IsActive {
			t.Error("an activity in use must be archived as inactive")
		}
	})

	t.Run("DELETE /api/v1/admin/games/:id refuses a teacher-owned activity", func(t *testing.T) {
		req := bearerRequest(http.MethodDelete, "/api/v1/admin/games/game-admin-1", nil, adminToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}

		var errRes struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &errRes); err != nil {
			t.Fatalf("failed to decode error envelope: %v", err)
		}
		if errRes.Error != "Solo se pueden gestionar actividades de la plataforma." {
			t.Errorf("unexpected error message: %q", errRes.Error)
		}

		stored, _ := repo.FindByID(context.Background(), "game-admin-1")
		if stored == nil {
			t.Fatal("teacher-owned activity must not be deleted")
		}
	})

	t.Run("teacher is rejected from the global activity CRUD (403)", func(t *testing.T) {
		teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
			ID:    "teacher-uuid-1",
			Email: "teacher@wordtap.app",
			Role:  entities.RoleInstructor,
		})

		body, _ := json.Marshal(map[string]any{"title": "No permitido"})
		req := bearerRequest(http.MethodPost, "/api/v1/admin/games", body, teacherToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected %d for teacher, got %d", http.StatusForbidden, rec.Code)
		}
	})
}
