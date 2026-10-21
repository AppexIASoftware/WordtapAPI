package entities_test

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestNormalizeLevel(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"B1 Intermediate", "b1", true},
		{"c2", "c2", true},
		{"upper-intermediate", "upper intermediate", true},
		{"Beginner", "beginner", true},
		{"  A2 Elementary  ", "a2", true},
		{"proficient", "proficient", true},
		{"", "", false},
		{"nivel desconocido", "", false},
	}

	for _, tc := range cases {
		got, ok := entities.NormalizeLevel(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Errorf("NormalizeLevel(%q) = (%q, %v), esperado (%q, %v)", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCompareLevels(t *testing.T) {
	cases := []struct {
		name        string
		courseLevel string
		gameLevel   string
		want        entities.LevelCompatibility
	}{
		{"mismo nivel", "B1", "B1 Intermediate", entities.LevelEqual},
		{"alias descriptivo", "beginner", "A1", entities.LevelEqual},
		{"juego superior", "B1", "C1", entities.LevelHigher},
		{"juego inferior", "B2", "A2 Elementary", entities.LevelLower},
		{"curso sin nivel", "", "B1", entities.LevelUnknown},
		{"juego sin nivel", "B1", "", entities.LevelUnknown},
		{"nivel no reconocido", "B1", "experto", entities.LevelUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := entities.CompareLevels(tc.courseLevel, tc.gameLevel); got != tc.want {
				t.Errorf("CompareLevels(%q, %q) = %q, esperado %q", tc.courseLevel, tc.gameLevel, got, tc.want)
			}
		})
	}
}

func TestLevelWarningMessage(t *testing.T) {
	if entities.LevelWarningMessage(entities.LevelLower) == "" {
		t.Error("un nivel inferior debe generar una advertencia de repaso")
	}
	if entities.LevelWarningMessage(entities.LevelHigher) == "" {
		t.Error("un nivel superior debe explicar el bloqueo")
	}
	if msg := entities.LevelWarningMessage(entities.LevelEqual); msg != "" {
		t.Errorf("un nivel igual no debe advertir, obtenido %q", msg)
	}
	if msg := entities.LevelWarningMessage(entities.LevelUnknown); msg != "" {
		t.Errorf("un nivel desconocido no debe advertir, obtenido %q", msg)
	}
}
