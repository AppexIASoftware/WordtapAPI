package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ProgressionThreshold is the share of a level's free lessons a student must
// complete before the following CEFR level opens.
const ProgressionThreshold = 0.80

// GamesPerLevelMilestone is how many global games a student must finish per CEFR
// level to reach the next one. Per-game completion is not persisted yet, so the
// progression signal for games is the aggregate victory counter.
const GamesPerLevelMilestone = 3

// LockReasonProgression marks content that is gated by the student's own
// progress rather than by payment. Entitlement locks stay client-side.
const LockReasonProgression = "progression"

// LockStateDTO describes why a catalog entry is locked and what unlocks it.
type LockStateDTO struct {
	Reason   string `json:"reason"`
	Requires string `json:"requires,omitempty"`
}

// CatalogUnlockStateDTO maps content identifiers to their progression lock.
// Unlocked entries are omitted so the payload stays small.
type CatalogUnlockStateDTO struct {
	CurrentLevel string                  `json:"current_level"`
	Courses      map[string]LockStateDTO `json:"courses"`
	Lessons      map[string]LockStateDTO `json:"lessons"`
	Games        map[string]LockStateDTO `json:"games"`
}

// levelProgress aggregates free lessons of one CEFR level.
type levelProgress struct {
	total     int
	completed int
}

// mastery returns the completed share of a level, treating an empty level as
// fully mastered so a missing level can never deadlock progression.
func (p levelProgress) mastery() float64 {
	if p.total == 0 {
		return 1
	}
	return float64(p.completed) / float64(p.total)
}

// buildLevelProgress aggregates free, published lessons per CEFR rank.
func buildLevelProgress(courses []entities.Course, completed map[string]struct{}) map[int]levelProgress {
	byRank := make(map[int]levelProgress)
	for _, course := range courses {
		rank, ok := entities.LevelRank(course.Level)
		if !ok {
			continue
		}
		entry := byRank[rank]
		for _, lesson := range course.Lessons {
			if lesson.AccessTier != entities.AccessTierFree {
				continue
			}
			entry.total++
			if _, done := completed[lesson.ID]; done {
				entry.completed++
			}
		}
		byRank[rank] = entry
	}
	return byRank
}

// blockingLevel returns the lowest level below rank that is not yet mastered.
func blockingLevel(rank int, byRank map[int]levelProgress) (int, bool) {
	for r := 1; r < rank; r++ {
		if byRank[r].mastery() < ProgressionThreshold {
			return r, true
		}
	}
	return 0, false
}

type GetCatalogUnlockStateQuery struct {
	UserID string
}

type GetCatalogUnlockStateHandler struct {
	courseRepo repositories.CourseRepository
	gameRepo   repositories.TeacherGameRepository
	statsRepo  repositories.UserLearningStatsRepository
}

func NewGetCatalogUnlockStateHandler(
	courseRepo repositories.CourseRepository,
	gameRepo repositories.TeacherGameRepository,
	statsRepo repositories.UserLearningStatsRepository,
) *GetCatalogUnlockStateHandler {
	return &GetCatalogUnlockStateHandler{
		courseRepo: courseRepo,
		gameRepo:   gameRepo,
		statsRepo:  statsRepo,
	}
}

func (h *GetCatalogUnlockStateHandler) Handle(
	ctx context.Context,
	query GetCatalogUnlockStateQuery,
) (*CatalogUnlockStateDTO, error) {
	if query.UserID == "" {
		return nil, errors.New("el ID de usuario es obligatorio")
	}

	courses, err := h.courseRepo.FindPublished(ctx)
	if err != nil {
		return nil, err
	}

	games, err := h.gameRepo.FindGlobal(ctx)
	if err != nil {
		return nil, err
	}

	completedIDs, err := h.statsRepo.GetCompletedLessonIDs(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	stats, err := h.statsRepo.GetLearningStats(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	completed := make(map[string]struct{}, len(completedIDs))
	for _, id := range completedIDs {
		completed[id] = struct{}{}
	}

	byRank := buildLevelProgress(courses, completed)

	state := &CatalogUnlockStateDTO{
		CurrentLevel: entities.LevelLabel(reachedLevel(byRank)),
		Courses:      make(map[string]LockStateDTO),
		Lessons:      make(map[string]LockStateDTO),
		Games:        make(map[string]LockStateDTO),
	}

	for _, course := range courses {
		rank, ok := entities.LevelRank(course.Level)
		if !ok {
			continue
		}

		if blocker, blocked := blockingLevel(rank, byRank); blocked {
			requires := entities.LevelLabel(blocker)
			state.Courses[course.ID] = LockStateDTO{Reason: LockReasonProgression, Requires: requires}
			for _, lesson := range course.Lessons {
				state.Lessons[lesson.ID] = LockStateDTO{Reason: LockReasonProgression, Requires: requires}
			}
			continue
		}

		applyLessonChain(course.Lessons, completed, state.Lessons)
	}

	gamesCompleted := 0
	if stats != nil {
		gamesCompleted = stats.GamesCompleted
	}
	for _, game := range games {
		rank, ok := entities.LevelRank(game.Level)
		if !ok || rank <= 1 {
			continue
		}
		if gamesCompleted >= GamesPerLevelMilestone*(rank-1) {
			continue
		}
		state.Games[game.ID] = LockStateDTO{
			Reason:   LockReasonProgression,
			Requires: entities.LevelLabel(rank - 1),
		}
	}

	return state, nil
}

// applyLessonChain locks every free lesson that follows an uncompleted free
// lesson. Non-free lessons are left to the entitlement gate, so they never block
// the chain for a student who cannot access them.
func applyLessonChain(lessons []entities.Lesson, completed map[string]struct{}, locks map[string]LockStateDTO) {
	var previous *entities.Lesson
	for i := range lessons {
		lesson := lessons[i]
		if lesson.AccessTier != entities.AccessTierFree {
			continue
		}
		if previous != nil {
			if _, done := completed[previous.ID]; !done {
				locks[lesson.ID] = LockStateDTO{
					Reason:   LockReasonProgression,
					Requires: previous.Title,
				}
			}
		}
		previous = &lessons[i]
	}
}

// reachedLevel returns the highest CEFR rank whose lower levels are all
// mastered. It cannot advance past one level beyond the highest level that
// actually has content, so a catalog without B2 lessons never reports C2.
func reachedLevel(byRank map[int]levelProgress) int {
	highestContent := 0
	for rank, entry := range byRank {
		if entry.total > 0 && rank > highestContent {
			highestContent = rank
		}
	}

	limit := highestContent + 1
	if limit > 6 {
		limit = 6
	}
	if limit < 1 {
		limit = 1
	}

	reached := 1
	for rank := 2; rank <= limit; rank++ {
		if _, blocked := blockingLevel(rank, byRank); blocked {
			break
		}
		reached = rank
	}
	return reached
}
