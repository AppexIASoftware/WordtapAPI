package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	app "github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationprefs"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// This test creates and drops a uniquely named database. Both opt-ins are
// required; never point this at a server without explicit authorization.
func TestNotificationPreferencesDisposableMySQL(t *testing.T) {
	if testing.Short() {
		t.Skip("disposable MySQL integration test")
	}
	dsn := os.Getenv("WORDTAP_NOTIFICATION_DISPOSABLE_DSN")
	if dsn == "" || os.Getenv("WORDTAP_NOTIFICATION_ALLOW_CREATE_DB") != "1" {
		t.Skip("requires authorized disposable MySQL DSN and database-creation opt-in")
	}
	config, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.DBName = ""
	config.ParseTime = true
	admin, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	name := "wordtap_notification_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Errorf("drop disposable database: %v", err)
		}
	})
	config.DBName = name
	db, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, ddl := range []string{
		`CREATE TABLE users (id varchar(36) PRIMARY KEY) ENGINE=InnoDB`,
		`CREATE TABLE notification_preferences (user_id varchar(36) PRIMARY KEY, enabled boolean NOT NULL, daily_word_enabled boolean NOT NULL, reminder_enabled boolean NOT NULL, reminder_time time NOT NULL, timezone varchar(80) NOT NULL, updated_at datetime NULL) ENGINE=InnoDB`,
		`CREATE TABLE notification_schedules (id varchar(36) PRIMARY KEY, user_id varchar(36) NOT NULL, notification_type varchar(40) NOT NULL, local_time time NOT NULL, days_of_week varchar(30) NOT NULL, is_enabled boolean NOT NULL, created_at datetime NULL) ENGINE=InnoDB`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	artifact, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "sql", "notification_schedule_modes.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(artifact)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO users (id) VALUES (?)`, "test-user").Error; err != nil {
		t.Fatal(err)
	}
	repo := NewMySQLNotificationPreferences(db)
	value := func(v bool) *bool { return &v }
	update := func(mode string, enabled bool, clock string) app.Update {
		u := app.Update{Enabled: value(enabled), DailyWordEnabled: value(false), ReminderEnabled: value(false), Timezone: "UTC", Mode: mode}
		if mode == app.ModeCustom {
			u.Schedules = []app.ScheduleUpdate{{Time: clock, Days: []int{1, 7}, Enabled: value(false)}}
		}
		return u
	}
	ctx := context.Background()
	first, err := repo.Put(ctx, "test-user", update(app.ModeCustom, false, "09:00"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Enabled || first.DailyWordEnabled || first.ReminderEnabled || len(first.Schedules) != 1 || first.Schedules[0].Enabled {
		t.Fatalf("false values not saved: %#v", first)
	}
	random, err := repo.Put(ctx, "test-user", update(app.ModeRandom, true, ""))
	if err != nil {
		t.Fatal(err)
	}
	if !random.Enabled || random.Mode != app.ModeRandom || len(random.Schedules) != 1 || random.Schedules[0].Time != "09:00" {
		t.Fatalf("random mode lost custom schedule: %#v", random)
	}
	// A malformed legacy row causes the final read to fail, after the writes;
	// verify the transaction rolled those writes back.
	if err := db.Exec(`UPDATE notification_schedules SET days_of_week = 'invalid' WHERE user_id = ?`, "test-user").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Put(ctx, "test-user", update(app.ModeRandom, false, "")); err == nil {
		t.Fatal("expected rollback")
	}
	var enabled bool
	if err := db.Table("notification_preferences").Select("enabled").Where("user_id = ?", "test-user").Scan(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("failed update was not rolled back")
	}
	if err := db.Exec(`UPDATE notification_schedules SET days_of_week = '1,7' WHERE user_id = ?`, "test-user").Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.Put(ctx, "test-user", update(app.ModeCustom, i == 0, fmt.Sprintf("%02d:00", 10+i)))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	final, err := repo.Get(ctx, "test-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Schedules) != 1 || (final.Enabled && final.Schedules[0].Time != "10:00") || (!final.Enabled && final.Schedules[0].Time != "11:00") {
		t.Fatalf("interleaved writes: %#v", final)
	}
}
