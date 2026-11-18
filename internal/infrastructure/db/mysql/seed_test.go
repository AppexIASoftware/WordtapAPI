package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type seedStubState struct {
	mu           sync.Mutex
	queries      []string
	execs        []string
	hijackedCrs  bool
	userRoleSeed string
}

type seedStubConnector struct{ state *seedStubState }

func (c seedStubConnector) Connect(context.Context) (driver.Conn, error) {
	return &seedStubConn{state: c.state}, nil
}
func (c seedStubConnector) Driver() driver.Driver { return nil }

func openSeedStubGORM(t *testing.T, state *seedStubState) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(seedStubConnector{state: state})
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

type seedStubConn struct{ state *seedStubState }

func (c *seedStubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not used") }
func (c *seedStubConn) Close() error                        { return nil }
func (c *seedStubConn) Begin() (driver.Tx, error)           { return seedStubTx{}, nil }

func (c *seedStubConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	c.state.execs = append(c.state.execs, query)
	c.state.mu.Unlock()
	return driver.RowsAffected(1), nil
}

func (c *seedStubConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	c.state.queries = append(c.state.queries, query)
	hijacked := c.state.hijackedCrs
	roleSeed := c.state.userRoleSeed
	c.state.mu.Unlock()

	if strings.Contains(strings.ToLower(query), "count(*)") {
		return &seedStubRows{cols: []string{"count"}, data: [][]driver.Value{{int64(1)}}}, nil
	}
	if hijacked && strings.Contains(query, "FROM `courses`") && len(args) > 0 && args[0].Value == "ingles-basico-gratuito" {
		return &seedStubRows{
			cols: []string{"id", "slug", "owner_type", "created_by"},
			data: [][]driver.Value{{"crs-official-1", "ingles-basico-gratuito", "teacher", "teacher-99"}},
		}, nil
	}
	if strings.Contains(query, "FROM `courses`") {
		return &seedStubRows{
			cols: []string{"id", "slug", "owner_type", "created_by"},
			data: [][]driver.Value{{"crs-official-2", "curso-premium-wordtap", "platform", nil}},
		}, nil
	}
	if strings.Contains(query, "FROM `users`") {
		role := "instructor"
		if roleSeed != "" {
			role = roleSeed
		}
		return &seedStubRows{cols: []string{"id", "role"}, data: [][]driver.Value{{"teacher-99", role}}}, nil
	}
	return &seedStubRows{cols: []string{"id"}, data: [][]driver.Value{{"row-1"}}}, nil
}

type seedStubTx struct{}

func (seedStubTx) Commit() error   { return nil }
func (seedStubTx) Rollback() error { return nil }

type seedStubRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func (r *seedStubRows) Columns() []string { return r.cols }
func (r *seedStubRows) Close() error      { return nil }
func (r *seedStubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func TestInitialPlatformCoursesOwnershipAndSeedNormalization(t *testing.T) {
	courses := initialPlatformCourses()
	if len(courses) != 2 {
		t.Fatalf("expected 2 initial platform courses, got %d", len(courses))
	}
	for _, c := range courses {
		if c.OwnerType != entities.OwnerTypePlatform || c.CreatedBy != nil {
			t.Fatalf("course %s owner_type=%q created_by=%v; want platform/nil", c.Slug, c.OwnerType, c.CreatedBy)
		}
	}

	state := &seedStubState{hijackedCrs: true}
	if err := Seed(openSeedStubGORM(t, state)); err != nil {
		t.Fatalf("Seed returned error: %v", err)
	}

	var normalizedOfficial bool
	for _, sqlText := range state.execs {
		if strings.Contains(sqlText, "WHERE created_by IS NULL") {
			t.Fatalf("Seed executed forbidden blanket created_by hijack: %s", sqlText)
		}
		if strings.Contains(sqlText, "UPDATE `courses` SET") && strings.Contains(sqlText, "`owner_type`") {
			normalizedOfficial = true
		}
	}
	if !normalizedOfficial {
		t.Fatal("expected Seed to normalize hijacked official course back to platform ownership")
	}

	t.Run("production skips demo teacher seed and explicit seed never demotes moderator", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("INITIAL_TEACHER_EMAIL", "")
		t.Setenv("DEV_TEACHER_EMAIL", "")
		prodState := &seedStubState{}
		if err := Seed(openSeedStubGORM(t, prodState)); err != nil {
			t.Fatalf("production Seed error: %v", err)
		}
		for _, q := range prodState.queries {
			if strings.Contains(q, "FROM `users`") {
				t.Fatalf("production Seed queried demo teacher unexpectedly: %s", q)
			}
		}

		t.Setenv("INITIAL_TEACHER_EMAIL", "mod@wordtap.app")
		modState := &seedStubState{userRoleSeed: "moderator"}
		if err := Seed(openSeedStubGORM(t, modState)); err != nil {
			t.Fatalf("explicit teacher Seed error: %v", err)
		}
		for _, ex := range modState.execs {
			if strings.Contains(ex, "UPDATE `users` SET") {
				t.Fatalf("Seed silently overwrote moderator role: %s", ex)
			}
		}
	})
}
