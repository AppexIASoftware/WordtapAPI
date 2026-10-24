package entities

import (
	"testing"
	"time"
)

func TestApplySM2(t *testing.T) {
	uv := &UserVocabulary{
		EaseFactor:       2.50,
		IntervalDays:     1,
		RepetitionNumber: 0,
	}

	// 1st review: perfect recall (q=5)
	uv.ApplySM2(5)
	if uv.RepetitionNumber != 1 {
		t.Fatalf("expected repetition 1, got %d", uv.RepetitionNumber)
	}
	if uv.IntervalDays != 1 {
		t.Fatalf("expected interval 1, got %d", uv.IntervalDays)
	}
	if uv.EaseFactor != 2.60 {
		t.Fatalf("expected EF 2.60, got %f", uv.EaseFactor)
	}
	if uv.TimesSeen != 1 || uv.TimesCorrect != 1 {
		t.Fatalf("expected times_seen=1, times_correct=1, got seen=%d, correct=%d", uv.TimesSeen, uv.TimesCorrect)
	}

	// 2nd review: good recall (q=4)
	uv.ApplySM2(4)
	if uv.RepetitionNumber != 2 {
		t.Fatalf("expected repetition 2, got %d", uv.RepetitionNumber)
	}
	if uv.IntervalDays != 6 {
		t.Fatalf("expected interval 6, got %d", uv.IntervalDays)
	}
	if uv.EaseFactor != 2.60 { // 2.60 + (0.1 - 1*(0.08+0.02)) = 2.60
		t.Fatalf("expected EF 2.60, got %f", uv.EaseFactor)
	}

	// 3rd review: fail (q=1)
	uv.ApplySM2(1)
	if uv.RepetitionNumber != 0 {
		t.Fatalf("expected repetition reset to 0, got %d", uv.RepetitionNumber)
	}
	if uv.IntervalDays != 1 {
		t.Fatalf("expected interval reset to 1, got %d", uv.IntervalDays)
	}
	if uv.EaseFactor >= 2.60 {
		t.Fatalf("expected EF to decrease, got %f", uv.EaseFactor)
	}
	if uv.NextReviewDate == nil {
		t.Fatalf("expected NextReviewDate to be set")
	}
}

func TestRecordVictory(t *testing.T) {
	stats := &UserLearningStats{
		UserID:         "user-1",
		LivesRemaining: 5,
		MaxLives:       5,
	}

	// First game victory today
	now := time.Now()
	stats.RecordVictory(3, 4, now)

	if stats.TotalStars != 3 {
		t.Errorf("expected 3 stars, got %d", stats.TotalStars)
	}
	if stats.TotalPoints != 30 {
		t.Errorf("expected 30 points, got %d", stats.TotalPoints)
	}
	if stats.GamesCompleted != 1 {
		t.Errorf("expected 1 game, got %d", stats.GamesCompleted)
	}
	if stats.WordsLearned != 4 {
		t.Errorf("expected 4 words, got %d", stats.WordsLearned)
	}
	if stats.CurrentStreakDays != 1 {
		t.Errorf("expected streak 1, got %d", stats.CurrentStreakDays)
	}

	// Consecutive day victory (tomorrow)
	tomorrow := now.Add(24 * time.Hour)
	stats.RecordVictory(2, 5, tomorrow)

	if stats.TotalStars != 5 {
		t.Errorf("expected 5 stars, got %d", stats.TotalStars)
	}
	if stats.TotalPoints != 50 {
		t.Errorf("expected 50 points, got %d", stats.TotalPoints)
	}
	if stats.CurrentStreakDays != 2 {
		t.Errorf("expected streak 2, got %d", stats.CurrentStreakDays)
	}
	if stats.LongestStreakDays != 2 {
		t.Errorf("expected longest streak 2, got %d", stats.LongestStreakDays)
	}

	// Missed 2 days (reset streak)
	threeDaysLater := tomorrow.Add(72 * time.Hour)
	stats.RecordVictory(3, 2, threeDaysLater)

	if stats.CurrentStreakDays != 1 {
		t.Errorf("expected streak reset to 1, got %d", stats.CurrentStreakDays)
	}
	if stats.LongestStreakDays != 2 {
		t.Errorf("expected longest streak to remain 2, got %d", stats.LongestStreakDays)
	}
}

func TestDeductAndRefillLives(t *testing.T) {
	stats := &UserLearningStats{
		UserID:         "user-1",
		LivesRemaining: 2,
		MaxLives:       5,
	}

	if !stats.DeductLife() {
		t.Errorf("expected DeductLife to return true")
	}
	if stats.LivesRemaining != 1 {
		t.Errorf("expected 1 life, got %d", stats.LivesRemaining)
	}
	if stats.LastLifeLostAt == nil {
		t.Errorf("expected LastLifeLostAt to be set")
	}

	stats.DeductLife()
	if stats.LivesRemaining != 0 {
		t.Errorf("expected 0 lives, got %d", stats.LivesRemaining)
	}

	// When 0 lives remaining
	if stats.DeductLife() {
		t.Errorf("expected DeductLife to return false when at 0")
	}

	stats.RefillLives()
	if stats.LivesRemaining != 5 {
		t.Errorf("expected 5 lives after refill, got %d", stats.LivesRemaining)
	}
	if stats.LastLifeLostAt != nil {
		t.Errorf("expected LastLifeLostAt to be nil after refill")
	}
}
