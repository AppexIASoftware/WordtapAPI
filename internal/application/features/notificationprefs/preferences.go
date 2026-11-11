package notification

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

const (
	ModeRandom = "random"
	ModeCustom = "custom"
)

var ErrInvalid = errors.New("invalid notification preferences")
var clockPattern = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)

type Schedule struct {
	Time    string `json:"time"`
	Days    []int  `json:"days"`
	Enabled bool   `json:"enabled"`
}

type ScheduleUpdate struct {
	Time    string `json:"time"`
	Days    []int  `json:"days"`
	Enabled *bool  `json:"enabled"`
}

type Preferences struct {
	Enabled          bool       `json:"enabled"`
	DailyWordEnabled bool       `json:"daily_word_enabled"`
	ReminderEnabled  bool       `json:"reminder_enabled"`
	Timezone         string     `json:"timezone"`
	Mode             string     `json:"mode"`
	Schedules        []Schedule `json:"schedules"`
	RandomCount      int        `json:"random_count"`
	RandomStart      string     `json:"random_start"`
	RandomEnd        string     `json:"random_end"`
}

// Update is a complete replacement of preferences; schedules are replaced only
// in custom mode, so toggling random does not discard previously saved times.
type Update struct {
	Enabled          *bool            `json:"enabled"`
	DailyWordEnabled *bool            `json:"daily_word_enabled"`
	ReminderEnabled  *bool            `json:"reminder_enabled"`
	Timezone         string           `json:"timezone"`
	Mode             string           `json:"mode"`
	Schedules        []ScheduleUpdate `json:"schedules"`
	RandomCount      *int             `json:"random_count"`
	RandomStart      *string          `json:"random_start"`
	RandomEnd        *string          `json:"random_end"`
}

type Repository interface {
	Get(context.Context, string) (*Preferences, error)
	Put(context.Context, string, Update) (*Preferences, error)
}

func (u Update) Validate() error {
	if u.Enabled == nil || u.DailyWordEnabled == nil || u.ReminderEnabled == nil {
		return fmt.Errorf("%w: all enabled fields are required", ErrInvalid)
	}
	if len(u.Timezone) == 0 || len(u.Timezone) > 80 || strings.TrimSpace(u.Timezone) != u.Timezone || u.Timezone == "Local" {
		return fmt.Errorf("%w: invalid timezone", ErrInvalid)
	}
	if _, err := time.LoadLocation(u.Timezone); err != nil {
		return fmt.Errorf("%w: invalid timezone", ErrInvalid)
	}
	if u.Mode != ModeRandom && u.Mode != ModeCustom {
		return fmt.Errorf("%w: invalid mode", ErrInvalid)
	}
	if (u.RandomCount != nil && *u.RandomCount != 3) || (u.RandomStart != nil && *u.RandomStart != "07:00") || (u.RandomEnd != nil && *u.RandomEnd != "22:00") {
		return fmt.Errorf("%w: random parameters are read-only", ErrInvalid)
	}
	for _, s := range u.Schedules {
		if s.Enabled == nil {
			return fmt.Errorf("%w: schedule enabled is required", ErrInvalid)
		}
	}
	if u.Mode == ModeRandom {
		return nil
	}
	if len(u.Schedules) < 1 || len(u.Schedules) > 5 {
		return fmt.Errorf("%w: custom mode requires one to five schedules", ErrInvalid)
	}
	seen := map[string]bool{}
	for _, s := range u.Schedules {
		if !clockPattern.MatchString(s.Time) || seen[s.Time] {
			return fmt.Errorf("%w: invalid or duplicate time", ErrInvalid)
		}
		seen[s.Time] = true
		if len(s.Days) == 0 || len(s.Days) > 7 {
			return fmt.Errorf("%w: invalid days", ErrInvalid)
		}
		days := map[int]bool{}
		for _, day := range s.Days {
			if day < 1 || day > 7 || days[day] {
				return fmt.Errorf("%w: invalid days", ErrInvalid)
			}
			days[day] = true
		}
	}
	return nil
}
