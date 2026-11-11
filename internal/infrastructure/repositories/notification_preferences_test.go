package repositories

import (
	"reflect"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestDecodeLegacySchedule(t *testing.T) {
	for _, tt := range []struct {
		name, clock, days, wantTime string
		enabled, invalid            bool
		wantDays                    []int
	}{
		{"standard", "09:00:00", "1,2,7", "09:00", false, false, []int{1, 2, 7}},
		{"retain seconds", "09:00:30", "1, 7", "09:00:30", true, false, []int{1, 7}},
		{"already short", "09:00", "5", "09:00", true, false, []int{5}},
		{"invalid stored days", "09:00:00", "0", "", false, true, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeLegacySchedule(entities.NotificationSchedule{LocalTime: tt.clock, DaysOfWeek: tt.days, IsEnabled: tt.enabled})
			if (err != nil) != tt.invalid {
				t.Fatalf("error = %v", err)
			}
			if !tt.invalid && (got.Time != tt.wantTime || got.Enabled != tt.enabled || !reflect.DeepEqual(got.Days, tt.wantDays)) {
				t.Fatalf("schedule = %#v", got)
			}
		})
	}
}
