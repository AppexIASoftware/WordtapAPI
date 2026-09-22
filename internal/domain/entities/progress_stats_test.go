package entities

import (
	"testing"
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
