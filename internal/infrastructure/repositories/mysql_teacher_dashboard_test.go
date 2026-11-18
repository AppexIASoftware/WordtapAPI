package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type stubSQLState struct {
	mu      sync.Mutex
	queries []string
	failOn  string
}

type stubConnector struct{ state *stubSQLState }

func (c stubConnector) Connect(context.Context) (driver.Conn, error) { return &stubConn{state: c.state}, nil }
func (c stubConnector) Driver() driver.Driver                        { return nil }

func openStubMySQLGORM(t *testing.T, state *stubSQLState) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(stubConnector{state: state})
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open stub: %v", err)
	}
	return db
}

type stubConn struct{ state *stubSQLState }

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not used") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error)           { return stubTx{}, nil }

func (c *stubConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	c.state.queries = append(c.state.queries, query)
	failOn := c.state.failOn
	c.state.mu.Unlock()

	if failOn != "" && strings.Contains(query, failOn) {
		return nil, errors.New("forced query failure on " + failOn)
	}
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "total_students"):
		return &stubRows{cols: []string{"total_students", "avg_streak", "retained"}, data: [][]driver.Value{{int64(10), 4.5, int64(6)}}}, nil
	case strings.Contains(lower, "vault_items"):
		return &stubRows{cols: []string{"vault_items", "vault_banks"}, data: [][]driver.Value{{int64(25), int64(9)}}}, nil
	case strings.Contains(lower, "active_subs"):
		return &stubRows{cols: []string{"total_cents", "count", "active_subs"}, data: [][]driver.Value{{int64(9900), int64(5), int64(3)}}}, nil
	case strings.Contains(lower, "total_cents"):
		return &stubRows{cols: []string{"total_cents", "count"}, data: [][]driver.Value{{int64(2500), int64(2)}}}, nil
	case strings.Contains(lower, "count("):
		return &stubRows{cols: []string{"count"}, data: [][]driver.Value{{int64(4)}}}, nil
	default:
		return &stubRows{cols: []string{"id"}, data: nil}, nil
	}
}

type stubTx struct{}

func (stubTx) Commit() error   { return nil }
func (stubTx) Rollback() error { return nil }

type stubRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func (r *stubRows) Columns() []string { return r.cols }
func (r *stubRows) Close() error      { return nil }
func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func TestMySQLTeacherDashboard_InstructorScopingAndRoundTripBudget(t *testing.T) {
	state := &stubSQLState{}
	res, err := NewMySQLDashboardRepository(openStubMySQLGORM(t, state)).
		WithSimulationEnabled(true).
		GetTeacherDashboard(context.Background(), "inst-42", "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.queries) != 4 {
		t.Fatalf("expected exactly 4 SQL round-trips, got %d", len(state.queries))
	}
	if res.TotalStudents != 10 || res.ActiveStudents != 4 || res.AvgStreakDays != 4.5 || res.RetentionPercent != 60.0 {
		t.Fatalf("unexpected student KPIs: %+v", res)
	}
	if res.VaultItems != 25 || res.VaultBanks != 9 || res.ActiveSubscriptions != 0 || res.RevenueCents != 2500 || res.PaidOrders != 2 {
		t.Fatalf("unexpected vault/billing KPIs: %+v", res)
	}

	var sawVaultCreatedBy, sawStudentScope, sawInstructorPurchases bool
	for _, sqlText := range state.queries {
		if strings.Contains(sqlText, "is_official") {
			t.Fatalf("obsolete column is_official used: %s", sqlText)
		}
		if strings.Contains(sqlText, "FROM orders") || strings.Contains(sqlText, "FROM subscriptions") {
			t.Fatalf("platform billing table queried for instructor scope: %s", sqlText)
		}
		if strings.Contains(sqlText, "content_bank_items") && strings.Contains(sqlText, "created_by IS NULL OR created_by = ?") {
			sawVaultCreatedBy = true
		}
		if strings.Contains(sqlText, "classroom_enrollments") && strings.Contains(sqlText, "cl.teacher_id = ?") && strings.Contains(sqlText, "c.owner_type = ?") {
			sawStudentScope = true
		}
		if strings.Contains(sqlText, "simulated_course_purchases") && strings.Contains(sqlText, "c.created_by = ?") && strings.Contains(sqlText, "c.owner_type = ?") {
			sawInstructorPurchases = true
		}
	}
	if !sawVaultCreatedBy || !sawStudentScope || !sawInstructorPurchases {
		t.Fatalf("missing expected scoped queries: vault=%v students=%v purchases=%v", sawVaultCreatedBy, sawStudentScope, sawInstructorPurchases)
	}

	// Production (simulationEnabled=false) skips simulated_course_purchases entirely (3 SQL round-trips).
	prodState := &stubSQLState{}
	prodRes, err := NewMySQLDashboardRepository(openStubMySQLGORM(t, prodState)).GetTeacherDashboard(context.Background(), "inst-42", "7d")
	if err != nil {
		t.Fatalf("unexpected error with simulation disabled: %v", err)
	}
	if len(prodState.queries) != 3 || prodRes.RevenueCents != 0 || prodRes.PaidOrders != 0 {
		t.Fatalf("expected 3 queries and 0 simulated revenue when simulation disabled, got queries=%d res=%+v", len(prodState.queries), prodRes)
	}
}

func TestMySQLTeacherDashboard_AdminGlobalAndErrorPropagation(t *testing.T) {
	state := &stubSQLState{}
	res, err := NewMySQLDashboardRepository(openStubMySQLGORM(t, state)).GetTeacherDashboard(context.Background(), "", "30d")
	if err != nil {
		t.Fatalf("unexpected error for admin scope: %v", err)
	}
	if len(state.queries) != 4 {
		t.Fatalf("expected exactly 4 SQL round-trips, got %d", len(state.queries))
	}
	if res.RevenueCents != 9900 || res.PaidOrders != 5 || res.ActiveSubscriptions != 3 {
		t.Fatalf("unexpected admin billing KPIs: %+v", res)
	}

	var sawOfficialOnly, sawBilling bool
	for _, sqlText := range state.queries {
		if strings.Contains(sqlText, "is_official") || strings.Contains(sqlText, "classroom_enrollments") {
			t.Fatalf("unexpected query in admin scope: %s", sqlText)
		}
		if strings.Contains(sqlText, "content_bank_items") && strings.Contains(sqlText, "created_by IS NULL") && !strings.Contains(sqlText, "OR created_by =") {
			sawOfficialOnly = true
		}
		if strings.Contains(sqlText, "FROM orders") && strings.Contains(sqlText, "FROM subscriptions") {
			sawBilling = true
		}
	}
	if !sawOfficialOnly || !sawBilling {
		t.Fatalf("missing expected admin queries: officialOnly=%v billing=%v", sawOfficialOnly, sawBilling)
	}

	if _, err := NewMySQLDashboardRepository(openStubMySQLGORM(t, &stubSQLState{failOn: "content_bank_items"})).GetTeacherDashboard(context.Background(), "inst-42", "7d"); err == nil {
		t.Fatal("expected error from core vault query failure, got nil")
	}
}
