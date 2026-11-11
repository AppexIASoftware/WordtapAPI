package inbox

import (
	"testing"
	"time"
)

func TestCursor(t *testing.T) {
	want := Cursor{ID: "b2a3a528-4591-4bee-bfbb-88e06965d87c", CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 123000, time.UTC)}
	got, err := DecodeCursor(EncodeCursor(want))
	if err != nil || got.ID != want.ID || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("cursor=%+v error=%v", got, err)
	}
	for _, bad := range []string{"!", "e30", EncodeCursor(Cursor{ID: "no", CreatedAt: time.Now()}), EncodeCursor(Cursor{ID: want.ID})} {
		if _, err := DecodeCursor(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if c, err := DecodeCursor(""); err != nil || c != nil {
		t.Fatalf("empty cursor=%v err=%v", c, err)
	}
}
