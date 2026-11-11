package repositories

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type inboxSQLLog struct{ statements []string }

func (l *inboxSQLLog) LogMode(logger.LogLevel) logger.Interface      { return l }
func (l *inboxSQLLog) Info(context.Context, string, ...interface{})  {}
func (l *inboxSQLLog) Warn(context.Context, string, ...interface{})  {}
func (l *inboxSQLLog) Error(context.Context, string, ...interface{}) {}
func (l *inboxSQLLog) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.statements = append(l.statements, sql)
}

func TestInboxQueriesScopeOwnerAndOrdering(t *testing.T) {
	// sql.Open only constructs a pool; DryRun and SkipInitializeWithVersion keep it offline.
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
	repo := NewMySQLNotificationInbox(db)
	ctx := context.Background()
	_, err = repo.List(ctx, "owner-a", 10, &inbox.Cursor{CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ID: "b2a3a528-4591-4bee-bfbb-88e06965d87c"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UnreadCount(ctx, "owner-a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.MarkRead(ctx, "owner-a", "b2a3a528-4591-4bee-bfbb-88e06965d87c")
	if err != nil {
		t.Fatal(err)
	}
	if len(log.statements) != 4 {
		t.Fatalf("statements=%v", log.statements)
	}
	for _, stmt := range log.statements {
		if !strings.Contains(stmt, "user_id = 'owner-a'") {
			t.Fatalf("unscoped query: %s", stmt)
		}
	}
	if !strings.Contains(log.statements[0], "ORDER BY created_at DESC,id DESC") || !strings.Contains(log.statements[0], "LIMIT 11") {
		t.Fatalf("pagination query: %s", log.statements[0])
	}
	if !strings.Contains(log.statements[2], "is_read = false") || !strings.Contains(log.statements[3], "id = ") {
		t.Fatalf("read queries: %v", log.statements[2:])
	}
}
