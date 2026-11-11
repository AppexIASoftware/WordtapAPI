package notification

import "testing"

func boolValue(v bool) *bool { return &v }

func TestUpdateValidation(t *testing.T) {
	valid := func() Update {
		return Update{Enabled: boolValue(false), DailyWordEnabled: boolValue(false), ReminderEnabled: boolValue(false), Timezone: "America/Argentina/Buenos_Aires", Mode: ModeCustom, Schedules: []ScheduleUpdate{{Time: "07:00", Days: []int{1, 7}, Enabled: boolValue(false)}}}
	}
	tests := []struct {
		name      string
		modify    func(*Update)
		wantError bool
	}{
		{"false values accepted", func(*Update) {}, false},
		{"missing boolean", func(u *Update) { u.Enabled = nil }, true},
		{"unknown timezone", func(u *Update) { u.Timezone = "Mars/Olympus" }, true},
		{"local timezone rejected", func(u *Update) { u.Timezone = "Local" }, true},
		{"timezone whitespace", func(u *Update) { u.Timezone = " UTC" }, true},
		{"invalid time", func(u *Update) { u.Schedules[0].Time = "24:00" }, true},
		{"seconds rejected", func(u *Update) { u.Schedules[0].Time = "07:00:00" }, true},
		{"duplicate time", func(u *Update) { u.Schedules = append(u.Schedules, u.Schedules[0]) }, true},
		{"duplicate days", func(u *Update) { u.Schedules[0].Days = []int{1, 1} }, true},
		{"invalid day", func(u *Update) { u.Schedules[0].Days = []int{0} }, true},
		{"empty custom", func(u *Update) { u.Schedules = nil }, true},
		{"six schedules", func(u *Update) { u.Schedules = make([]ScheduleUpdate, 6) }, true},
		{"random", func(u *Update) { u.Mode = ModeRandom; u.Schedules = nil }, false},
		{"random retains submitted schedules", func(u *Update) { u.Mode = ModeRandom }, false},
		{"missing schedule enabled", func(u *Update) { u.Schedules[0].Enabled = nil }, true},
		{"unknown mode", func(u *Update) { u.Mode = "other" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := valid()
			tt.modify(&u)
			if got := u.Validate() != nil; got != tt.wantError {
				t.Fatalf("error = %v, want error %v", got, tt.wantError)
			}
		})
	}
}
