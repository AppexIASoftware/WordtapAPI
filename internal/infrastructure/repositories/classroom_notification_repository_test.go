package repositories

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestClassroomNotificationScopeUsesClassroomSchema(t *testing.T) {
	pool, err := sql.Open("mysql", "test:test@tcp(127.0.0.1:1)/unused")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	log := &inboxSQLLog{}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, SkipDefaultTransaction: true, DisableAutomaticPing: true, Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMySQLClassroomNotificationRepository(db)
	if _, err = repo.ListByClassroomWithCourseScope(context.Background(), "class-1", 20, 0); err != nil {
		t.Fatal(err)
	}
	if len(log.statements) == 0 || strings.Contains(log.statements[0], "classroom_courses") {
		t.Fatalf("expected classroom schema, got SQL %v", log.statements)
	}
	if _, err = repo.CountByClassroomWithCourseScope(context.Background(), "class-1"); err != nil {
		t.Fatal(err)
	}
	if len(log.statements) != 4 {
		t.Fatalf("expected classroom lookup and notification query for list and count: %v", log.statements)
	}
	for _, i := range []int{1, 3} {
		if !strings.Contains(log.statements[i], "classroom_id = 'class-1'") || !strings.Contains(log.statements[i], "target_course_id IS NULL OR target_course_id IN") {
			t.Fatalf("missing classroom or course scope: %s", log.statements[i])
		}
	}
}

func TestClassroomHistoryFiltersListAndCount(t *testing.T) {
	pool, err := sql.Open("mysql", "test:test@tcp(127.0.0.1:1)/unused")
	if err != nil { t.Fatal(err) }
	defer pool.Close()
	log := &inboxSQLLog{}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, SkipDefaultTransaction: true, DisableAutomaticPing: true, Logger: log})
	if err != nil { t.Fatal(err) }
	status := entities.ClassroomNotificationStatusFailed
	from, to := "2026-01-02T00:00:00Z", "2026-01-03T00:00:00Z"
	f := domainRepo.ClassroomNotificationFilter{Status: &status, DateFrom: &from, DateTo: &to}
	repo := NewMySQLClassroomNotificationRepository(db)
	if _, err := repo.ListByClassroomWithCourseScope(context.Background(), "class-1", 20, 0, f); err != nil { t.Fatal(err) }
	if _, err := repo.CountByClassroomWithCourseScope(context.Background(), "class-1", f); err != nil { t.Fatal(err) }
	if len(log.statements) != 4 { t.Fatalf("statements: %v", log.statements) }
	for _, i := range []int{1, 3} {
		for _, condition := range []string{"classroom_id = 'class-1'", "target_course_id IS NULL OR target_course_id IN", "status = 'failed'", "sent_at >=", "sent_at <="} {
			if !strings.Contains(log.statements[i], condition) { t.Fatalf("missing %s: %s", condition, log.statements[i]) }
		}
	}
}
