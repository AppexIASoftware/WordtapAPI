package queries

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func courseFixture(id, level string, lessons ...entities.Lesson) entities.Course {
	return entities.Course{ID: id, Level: level, Lessons: lessons}
}

func lessonFixture(id, title string, tier entities.AccessTier) entities.Lesson {
	return entities.Lesson{ID: id, Title: title, AccessTier: tier}
}

func TestMasteryTreatsEmptyLevelAsMastered(t *testing.T) {
	if got := (levelProgress{}).mastery(); got != 1 {
		t.Fatalf("empty level mastery = %v, want 1", got)
	}
	if got := (levelProgress{total: 5, completed: 4}).mastery(); got < ProgressionThreshold {
		t.Fatalf("4/5 mastery = %v, want >= %v", got, ProgressionThreshold)
	}
	if got := (levelProgress{total: 5, completed: 3}).mastery(); got >= ProgressionThreshold {
		t.Fatalf("3/5 mastery = %v, want < %v", got, ProgressionThreshold)
	}
}

func TestBuildLevelProgressCountsOnlyFreeLessons(t *testing.T) {
	courses := []entities.Course{
		courseFixture("c-a1", "A1",
			lessonFixture("l1", "Uno", entities.AccessTierFree),
			lessonFixture("l2", "Dos", entities.AccessTierFree),
			lessonFixture("l3", "Pro", entities.AccessTierSubscription),
		),
	}

	byRank := buildLevelProgress(courses, map[string]struct{}{"l1": {}, "l3": {}})

	if byRank[1].total != 2 {
		t.Fatalf("A1 total = %d, want 2 (subscription lessons excluded)", byRank[1].total)
	}
	if byRank[1].completed != 1 {
		t.Fatalf("A1 completed = %d, want 1", byRank[1].completed)
	}
}

func TestBlockingLevelOpensNextLevelAtThreshold(t *testing.T) {
	almost := map[int]levelProgress{1: {total: 5, completed: 4}}
	if blocker, blocked := blockingLevel(2, almost); blocked {
		t.Fatalf("A2 blocked at %d, want unlocked at 80%%", blocker)
	}

	short := map[int]levelProgress{1: {total: 5, completed: 3}}
	blocker, blocked := blockingLevel(2, short)
	if !blocked || blocker != 1 {
		t.Fatalf("A2 blocker = (%d, %v), want (1, true)", blocker, blocked)
	}
}

func TestBlockingLevelIgnoresLevelsWithoutFreeContent(t *testing.T) {
	byRank := map[int]levelProgress{1: {total: 2, completed: 2}}
	if _, blocked := blockingLevel(4, byRank); blocked {
		t.Fatal("B2 should stay unlocked when A2 and B1 have no free lessons")
	}
}

func TestApplyLessonChainLocksLessonsAfterUncompletedFreeLesson(t *testing.T) {
	lessons := []entities.Lesson{
		lessonFixture("l1", "Primera", entities.AccessTierFree),
		lessonFixture("l2", "Segunda", entities.AccessTierFree),
		lessonFixture("l3", "Tercera", entities.AccessTierFree),
	}

	locks := make(map[string]LockStateDTO)
	applyLessonChain(lessons, map[string]struct{}{"l1": {}}, locks)

	if _, locked := locks["l1"]; locked {
		t.Fatal("first lesson must never be progression-locked")
	}
	if _, locked := locks["l2"]; locked {
		t.Fatal("l2 must be unlocked once the previous lesson is completed")
	}
	if lock, locked := locks["l3"]; !locked || lock.Requires != "Segunda" {
		t.Fatalf("l3 lock = %+v, want requires Segunda", lock)
	}
}

func TestApplyLessonChainIgnoresNonFreeLessons(t *testing.T) {
	lessons := []entities.Lesson{
		lessonFixture("l1", "Primera", entities.AccessTierFree),
		lessonFixture("pro", "Premium", entities.AccessTierSubscription),
		lessonFixture("l2", "Segunda", entities.AccessTierFree),
	}

	locks := make(map[string]LockStateDTO)
	applyLessonChain(lessons, map[string]struct{}{"l1": {}}, locks)

	if _, locked := locks["pro"]; locked {
		t.Fatal("subscription lessons belong to the entitlement gate, not the chain")
	}
	if _, locked := locks["l2"]; locked {
		t.Fatal("a pro lesson must not block the free chain")
	}
}

func TestReachedLevelAdvancesOnlyWithMastery(t *testing.T) {
	if got := reachedLevel(map[int]levelProgress{1: {total: 2, completed: 2}}); got != 2 {
		t.Fatalf("reachedLevel = %d, want 2", got)
	}
	if got := reachedLevel(map[int]levelProgress{1: {total: 4, completed: 1}}); got != 1 {
		t.Fatalf("reachedLevel = %d, want 1", got)
	}
}

func TestLevelLabelsRoundTrip(t *testing.T) {
	for rank := 1; rank <= 6; rank++ {
		label := entities.LevelLabel(rank)
		gotRank, ok := entities.LevelRank(label)
		if !ok || gotRank != rank {
			t.Fatalf("LevelRank(%q) = (%d, %v), want (%d, true)", label, gotRank, ok, rank)
		}
	}
	if _, ok := entities.LevelRank("beginner"); !ok {
		t.Fatal("legacy 'beginner' alias must keep a rank")
	}
}
