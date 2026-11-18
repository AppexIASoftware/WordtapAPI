package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type recordedQuery struct {
	sql  string
	args []driver.Value
}

type queryResponse struct {
	columns []string
	rows    [][]driver.Value
}

type execResponse struct {
	rowsAffected int64
	err          error
}

type mockDBState struct {
	events      []string
	queries     []recordedQuery
	execs       []recordedQuery
	onQuery     func(query string, args []driver.Value) (queryResponse, bool)
	onExec      func(query string, args []driver.Value) (execResponse, bool)
	beginErr    error
	commitErr   error
	rollbackErr error
}

type mockConnector struct{ state *mockDBState }

func (c *mockConnector) Connect(context.Context) (driver.Conn, error) {
	return &mockConn{state: c.state}, nil
}
func (c *mockConnector) Driver() driver.Driver { return &mockDriver{state: c.state} }

type mockDriver struct{ state *mockDBState }

func (d *mockDriver) Open(string) (driver.Conn, error) { return &mockConn{state: d.state}, nil }

type mockConn struct{ state *mockDBState }

func (c *mockConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *mockConn) Close() error                        { return nil }
func (c *mockConn) Begin() (driver.Tx, error)           { return c.BeginTx(context.Background(), driver.TxOptions{}) }
func (c *mockConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.state.events = append(c.state.events, "BEGIN")
	if c.state.beginErr != nil {
		return nil, c.state.beginErr
	}
	return &mockTx{state: c.state}, nil
}

func (c *mockConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	vals := extractNamedValues(args)
	c.state.events = append(c.state.events, "EXEC:"+query)
	c.state.execs = append(c.state.execs, recordedQuery{sql: query, args: vals})
	if c.state.onExec != nil {
		if resp, ok := c.state.onExec(query, vals); ok {
			if resp.err != nil {
				return nil, resp.err
			}
			return driver.RowsAffected(resp.rowsAffected), nil
		}
	}
	return driver.RowsAffected(1), nil
}

func (c *mockConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	vals := extractNamedValues(args)
	c.state.events = append(c.state.events, "QUERY:"+query)
	c.state.queries = append(c.state.queries, recordedQuery{sql: query, args: vals})
	if c.state.onQuery != nil {
		if resp, ok := c.state.onQuery(query, vals); ok {
			return &mockRows{columns: resp.columns, rows: resp.rows}, nil
		}
	}
	return &mockRows{}, nil
}

type mockTx struct{ state *mockDBState }

func (t *mockTx) Commit() error {
	t.state.events = append(t.state.events, "COMMIT")
	return t.state.commitErr
}
func (t *mockTx) Rollback() error {
	t.state.events = append(t.state.events, "ROLLBACK")
	return t.state.rollbackErr
}

type mockRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *mockRows) Columns() []string { return r.columns }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func extractNamedValues(named []driver.NamedValue) []driver.Value {
	out := make([]driver.Value, len(named))
	for i, n := range named {
		out[i] = n.Value
	}
	return out
}

func newLockingTestGormDB(t *testing.T, state *mockDBState) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(&mockConnector{state: state})
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open mock gorm DB: %v", err)
	}
	return db
}

func hasPrefixEvent(events []string, prefix string) bool {
	for _, ev := range events {
		if strings.HasPrefix(ev, prefix) {
			return true
		}
	}
	return false
}

func hasQueryContaining(queries []recordedQuery, sub string) bool {
	for _, q := range queries {
		if strings.Contains(q.sql, sub) {
			return true
		}
	}
	return false
}

func TestUserStatsTransactionsAndLocking(t *testing.T) {
	now := time.Now().UTC()

	t.Run("DeductUserLife locks inside transaction and uses UTC", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`user_learning_stats`") {
					return queryResponse{
						columns: []string{"user_id", "lives_remaining", "max_lives", "updated_at"},
						rows:    [][]driver.Value{{"user-1", int64(5), int64(5), now}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		repo := &MySQLUserRepository{db: newLockingTestGormDB(t, state)}

		stats, err := repo.DeductUserLife(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.LivesRemaining != 4 {
			t.Fatalf("expected 4 lives remaining, got %d", stats.LivesRemaining)
		}
		if stats.LastLifeLostAt == nil || stats.LastLifeLostAt.Location() != time.UTC {
			t.Fatalf("expected UTC LastLifeLostAt, got %v", stats.LastLifeLostAt)
		}
		if !hasPrefixEvent(state.events, "BEGIN") || !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected BEGIN/COMMIT events, got %v", state.events)
		}
		if !hasQueryContaining(state.queries, "FOR UPDATE") {
			t.Fatalf("expected FOR UPDATE query, got %+v", state.queries)
		}
	})

	t.Run("RefillUserLives locks inside transaction and resets lives", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`user_learning_stats`") {
					return queryResponse{
						columns: []string{"user_id", "lives_remaining", "max_lives", "last_life_lost_at"},
						rows:    [][]driver.Value{{"user-1", int64(1), int64(5), now}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		repo := &MySQLUserRepository{db: newLockingTestGormDB(t, state)}

		stats, err := repo.RefillUserLives(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.LivesRemaining != 5 || stats.LastLifeLostAt != nil {
			t.Fatalf("expected refilled lives=5 and nil LastLifeLostAt, got %+v", stats)
		}
		if !hasQueryContaining(state.queries, "FOR UPDATE") || !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected FOR UPDATE and COMMIT, events=%v", state.events)
		}
	})

	t.Run("RecordGameVictory seeds OnConflict before FOR UPDATE locks", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				switch {
				case strings.Contains(q, "`user_learning_stats`"):
					return queryResponse{
						columns: []string{"user_id", "lives_remaining", "max_lives", "total_stars", "total_points", "games_completed", "lessons_completed"},
						rows:    [][]driver.Value{{"user-1", int64(5), int64(5), int64(3), int64(30), int64(1), int64(1)}},
					}, true
				case strings.Contains(q, "`user_lesson_progresses`"):
					return queryResponse{
						columns: []string{"id", "user_id", "lesson_id", "completed_items", "total_items", "progress_percent"},
						rows:    [][]driver.Value{{"prog-1", "user-1", "lesson-1", int64(3), int64(10), float64(30)}},
					}, true
				case strings.Contains(q, "`user_daily_activities`"):
					return queryResponse{
						columns: []string{"id", "user_id", "activity_date", "minutes_studied", "questions_answered", "correct_answers", "points_earned"},
						rows:    [][]driver.Value{{"act-1", "user-1", now, int64(2), int64(5), int64(5), int64(30)}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		repo := &MySQLUserRepository{db: newLockingTestGormDB(t, state)}

		res, err := repo.RecordGameVictory(context.Background(), domainRepo.RecordProgressParams{
			UserID:         "user-1",
			LessonID:       "lesson-1",
			StarsEarned:    3,
			WordsPracticed: 4,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TotalStars != 6 || res.GamesCompleted != 2 || res.LessonsCompleted != 2 {
			t.Fatalf("unexpected result stats: %+v", res)
		}
		var progressUpdate *recordedQuery
		for i := range state.execs {
			if strings.Contains(state.execs[i].sql, "UPDATE `user_lesson_progresses`") {
				progressUpdate = &state.execs[i]
				break
			}
		}
		if progressUpdate == nil {
			t.Fatalf("expected UPDATE user_lesson_progresses statement, execs=%+v", state.execs)
		}
		has100Pct := false
		int10Count := 0
		for _, arg := range progressUpdate.args {
			if arg == float64(100) {
				has100Pct = true
			}
			if arg == int64(10) {
				int10Count++
			}
		}
		if !has100Pct || int10Count < 2 {
			t.Fatalf("expected progress_percent=100 and completed_items=total_items=10 after BeforeUpdate hook, args=%+v", progressUpdate.args)
		}
		lockCount := 0
		for _, q := range state.queries {
			if strings.Contains(q.sql, "FOR UPDATE") {
				lockCount++
			}
		}
		if lockCount != 3 {
			t.Fatalf("expected 3 FOR UPDATE queries (stats, lesson progress, daily activity), got %d", lockCount)
		}
		upsertCount := 0
		for _, ex := range state.execs {
			if strings.Contains(ex.sql, "ON DUPLICATE KEY UPDATE") || strings.Contains(ex.sql, "INSERT IGNORE") {
				upsertCount++
			}
		}
		if upsertCount != 3 {
			t.Fatalf("expected 3 OnConflict seed upserts before FOR UPDATE, got %d (execs=%+v)", upsertCount, state.execs)
		}
		if !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected COMMIT, events=%v", state.events)
		}
	})
}

func TestMarkReadUsesClassroomNotificationFKJoin(t *testing.T) {
	state := &mockDBState{
		onExec: func(_ string, _ []driver.Value) (execResponse, bool) {
			return execResponse{rowsAffected: 1}, true
		},
	}
	repo := NewMySQLNotificationInbox(newLockingTestGormDB(t, state))

	updated, err := repo.MarkRead(context.Background(), "user-1", "notif-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated=true")
	}
	if !hasPrefixEvent(state.events, "BEGIN") || !hasPrefixEvent(state.events, "COMMIT") {
		t.Fatalf("expected BEGIN/COMMIT around MarkRead updates, events=%v", state.events)
	}
	if len(state.execs) != 2 {
		t.Fatalf("expected 2 exec statements, got %d (%+v)", len(state.execs), state.execs)
	}
	joinSQL := state.execs[1].sql
	if !strings.Contains(joinSQL, "INNER JOIN notifications n ON n.classroom_notification_id = cn.id") {
		t.Fatalf("expected exact FK JOIN query, got: %s", joinSQL)
	}
	if strings.Contains(joinSQL, "BETWEEN") || strings.Contains(joinSQL, "title =") {
		t.Fatalf("legacy title/time heuristic should not appear in SQL: %s", joinSQL)
	}
}

func TestSyncReadCountsAggregatesByNotificationIDFK(t *testing.T) {
	now := time.Now().UTC()
	state := &mockDBState{
		onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
			switch {
			case strings.Contains(q, "`classroom_notifications`"):
				return queryResponse{
					columns: []string{"id", "teacher_id", "title", "message", "action_type", "sent_at", "recipient_count", "read_count", "status", "created_at"},
					rows: [][]driver.Value{
						{"cn-legacy", "teacher-1", "Legacy Notice", "Body", "open_classroom", now, int64(5), int64(3), "sent", now},
						{"cn-fk", "teacher-1", "FK Notice", "Body", "open_classroom", now, int64(2), int64(0), "failed", now},
					},
				}, true
			case strings.Contains(q, "classroom_notification_id"):
				return queryResponse{
					columns: []string{"classroom_notification_id", "total"},
					rows: [][]driver.Value{
						{"cn-legacy", int64(1)},
						{"cn-fk", int64(2)},
					},
				}, true
			}
			return queryResponse{}, false
		},
	}
	repo := NewMySQLClassroomNotificationRepository(newLockingTestGormDB(t, state))
	from, to := "2026-11-01T00:00:00Z", "2026-11-18T00:00:00Z"

	items, err := repo.ListAll(context.Background(), domainRepo.ClassroomNotificationFilter{
		DateFrom: &from,
		DateTo:   &to,
	}, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].ReadCount != 3 || items[1].ReadCount != 2 || items[1].Status != entities.ClassroomNotificationStatusSent {
		t.Fatalf("unexpected items state: %+v", items)
	}
	if !hasQueryContaining(state.queries, "GROUP BY `classroom_notification_id`") {
		t.Fatalf("expected GROUP BY classroom_notification_id query, got %+v", state.queries)
	}
	if len(state.queries[0].args) < 2 {
		t.Fatalf("expected parsed time filter args in ListAll query, got %+v", state.queries[0].args)
	}
	for i := 0; i < 2; i++ {
		parsed, ok := state.queries[0].args[i].(time.Time)
		if !ok || parsed.Location() != time.UTC {
			t.Fatalf("expected ListAll date arg[%d] as UTC time.Time, got %T (%v)", i, state.queries[0].args[i], state.queries[0].args[i])
		}
	}
}
