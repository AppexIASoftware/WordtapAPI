package entities

import "strings"

// LevelCompatibility describes CEFR relationship between course and game.
type LevelCompatibility string

const (
	LevelEqual   LevelCompatibility = "equal"
	LevelLower   LevelCompatibility = "lower"
	LevelHigher  LevelCompatibility = "higher"
	LevelUnknown LevelCompatibility = "unknown"
)

// levelRanks maps CEFR levels and standard aliases to comparable ranks.
var levelRanks = map[string]int{
	"a1": 1, "beginner": 1,
	"a2": 2, "elementary": 2,
	"b1": 3, "intermediate": 3,
	"b2": 4, "upper intermediate": 4,
	"c1": 5, "advanced": 5,
	"c2": 6, "proficient": 6,
}

// isCERLevel reports whether the prefix is a CEFR level (A1..C2).
func isCERLevel(prefix string) bool {
	if len(prefix) != 2 {
		return false
	}
	letter := prefix[0]
	digit := prefix[1]
	return letter >= 'a' && letter <= 'c' && (digit == '1' || digit == '2')
}

// NormalizeLevel converts a level string to its canonical key.
func NormalizeLevel(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.NewReplacer("-", " ", "_", " ").Replace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	if normalized == "" {
		return "", false
	}

	if len(normalized) >= 2 {
		prefix := normalized[:2]
		if isCERLevel(prefix) {
			return prefix, true
		}
	}

	if _, ok := levelRanks[normalized]; ok {
		return normalized, true
	}

	return "", false
}

// LevelRank returns the canonical CEFR rank (1 = A1 ... 6 = C2) for a level string.
func LevelRank(raw string) (int, bool) {
	key, ok := NormalizeLevel(raw)
	if !ok {
		return 0, false
	}
	rank, known := levelRanks[key]
	if !known {
		return 0, false
	}
	return rank, true
}

// LevelLabel returns the CEFR label for a canonical rank.
func LevelLabel(rank int) string {
	labels := [...]string{"A1", "A2", "B1", "B2", "C1", "C2"}
	if rank < 1 || rank > len(labels) {
		return ""
	}
	return labels[rank-1]
}

// CompareLevels compares course and game levels.
func CompareLevels(courseLevel, gameLevel string) LevelCompatibility {
	courseKey, courseOK := NormalizeLevel(courseLevel)
	gameKey, gameOK := NormalizeLevel(gameLevel)
	if !courseOK || !gameOK {
		return LevelUnknown
	}

	courseRank := levelRanks[courseKey]
	gameRank := levelRanks[gameKey]

	switch {
	case gameRank == courseRank:
		return LevelEqual
	case gameRank < courseRank:
		return LevelLower
	default:
		return LevelHigher
	}
}

// LevelWarningMessage returns warning message for level mismatch.
func LevelWarningMessage(compat LevelCompatibility) string {
	switch compat {
	case LevelLower:
		return "Esta actividad es de un nivel inferior al del curso; puedes usarla como repaso."
	case LevelHigher:
		return "El nivel de esta actividad supera el nivel del curso; elige una actividad acorde o ajusta el nivel del curso."
	default:
		return ""
	}
}
