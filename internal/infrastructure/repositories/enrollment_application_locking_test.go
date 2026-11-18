package repositories

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestEnrollStudentTransactionAndIdempotency(t *testing.T) {
	t.Run("returns existing locked enrollment idempotently without duplicate insert", func(t *testing.T) {
		now := time.Now().UTC()
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				switch {
				case strings.Contains(q, "`classrooms`"):
					return queryResponse{
						columns: []string{"id", "is_active"},
						rows:    [][]driver.Value{{"cls-1", true}},
					}, true
				case strings.Contains(q, "`classroom_enrollments`"):
					return queryResponse{
						columns: []string{"id", "classroom_id", "student_id", "enrolled_at", "accuracy_pct", "streak_days"},
						rows:    [][]driver.Value{{"enr-existing", "cls-1", "stu-1", now, int64(100), int64(1)}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		repo := NewMySQLClassroomRepository(newLockingTestGormDB(t, state))

		enrollment := &entities.ClassroomEnrollment{
			ClassroomID: "cls-1",
			StudentID:   "stu-1",
		}
		if err := repo.EnrollStudent(context.Background(), enrollment); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enrollment.ID != "enr-existing" {
			t.Fatalf("expected existing enrollment ID 'enr-existing', got %q", enrollment.ID)
		}
		if len(state.execs) != 0 {
			t.Fatalf("expected zero INSERT statements on idempotent re-enroll, got %+v", state.execs)
		}
		if !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected COMMIT, events=%v", state.events)
		}
	})

	t.Run("rolls back when locked classroom is inactive", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`classrooms`") {
					return queryResponse{
						columns: []string{"id", "is_active"},
						rows:    [][]driver.Value{{"cls-inactive", false}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		repo := NewMySQLClassroomRepository(newLockingTestGormDB(t, state))

		err := repo.EnrollStudent(context.Background(), &entities.ClassroomEnrollment{
			ClassroomID: "cls-inactive",
			StudentID:   "stu-1",
		})
		if err == nil {
			t.Fatal("expected error when classroom is inactive")
		}
		if !hasPrefixEvent(state.events, "ROLLBACK") {
			t.Fatalf("expected ROLLBACK on inactive classroom, events=%v", state.events)
		}
	})
}

func TestTeacherApplicationTransactionsAndRoleGuards(t *testing.T) {
	appAndUserRows := func(appStatus entities.ApplicationStatus, userRole entities.UserRole) func(string, []driver.Value) (queryResponse, bool) {
		return func(q string, _ []driver.Value) (queryResponse, bool) {
			switch {
			case strings.Contains(strings.ToLower(q), "count(*)") && strings.Contains(q, "`teacher_applications`"):
				return queryResponse{
					columns: []string{"count"},
					rows:    [][]driver.Value{{int64(0)}},
				}, true
			case strings.Contains(q, "`teacher_applications`"):
				return queryResponse{
					columns: []string{"id", "user_id", "status"},
					rows:    [][]driver.Value{{"app-1", "user-1", string(appStatus)}},
				}, true
			case strings.Contains(q, "`users`"):
				return queryResponse{
					columns: []string{"id", "role", "is_active"},
					rows:    [][]driver.Value{{"user-1", string(userRole), true}},
				}, true
			}
			return queryResponse{}, false
		}
	}

	t.Run("refuses admin account mutation in ApproveInTx", func(t *testing.T) {
		state := &mockDBState{onQuery: appAndUserRows(entities.ApplicationStatusPending, entities.RoleAdmin)}
		repo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, state))

		err := repo.ApproveInTx(context.Background(), "app-1", "reviewer-1")
		if err == nil || !strings.Contains(err.Error(), "administrator") {
			t.Fatalf("expected administrator guard error, got %v", err)
		}
		if !hasPrefixEvent(state.events, "ROLLBACK") {
			t.Fatalf("expected ROLLBACK, events=%v", state.events)
		}
	})

	t.Run("preserves moderator role when approving application", func(t *testing.T) {
		state := &mockDBState{onQuery: appAndUserRows(entities.ApplicationStatusPending, entities.RoleModerator)}
		repo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, state))

		if err := repo.ApproveInTx(context.Background(), "app-1", "reviewer-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, ex := range state.execs {
			if strings.Contains(ex.sql, "UPDATE `users`") {
				t.Fatalf("expected active moderator user not to be downgraded, got exec: %s", ex.sql)
			}
		}
		if !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected COMMIT, events=%v", state.events)
		}
	})

	t.Run("refuses moderator suspension and pending suspension", func(t *testing.T) {
		state := &mockDBState{onQuery: appAndUserRows(entities.ApplicationStatusApproved, entities.RoleModerator)}
		repo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, state))

		err := repo.SuspendInTx(context.Background(), "app-1", "reviewer-1", "policy")
		if err == nil || !strings.Contains(err.Error(), "moderator") {
			t.Fatalf("expected moderator suspension guard error, got %v", err)
		}

		pendingState := &mockDBState{onQuery: appAndUserRows(entities.ApplicationStatusPending, entities.RoleStudent)}
		pendingRepo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, pendingState))
		if err := pendingRepo.SuspendInTx(context.Background(), "app-1", "reviewer-1", "policy"); err == nil || !strings.Contains(err.Error(), "only approved") {
			t.Fatalf("expected only-approved suspension error, got %v", err)
		}
	})

	t.Run("RejectInTx locks FOR UPDATE and blocks rejecting approved or suspended applications", func(t *testing.T) {
		for _, status := range []entities.ApplicationStatus{entities.ApplicationStatusApproved, entities.ApplicationStatusSuspended} {
			state := &mockDBState{onQuery: appAndUserRows(status, entities.RoleInstructor)}
			repo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, state))

			err := repo.RejectInTx(context.Background(), "app-1", "reviewer-1", "too late")
			if err == nil || !strings.Contains(err.Error(), "approved or suspended") {
				t.Fatalf("expected approved/suspended guard error for %s, got %v", status, err)
			}
			if !hasQueryContaining(state.queries, "FOR UPDATE") || !hasPrefixEvent(state.events, "ROLLBACK") {
				t.Fatalf("expected FOR UPDATE and ROLLBACK for %s, events=%v", status, state.events)
			}
		}
	})

	t.Run("DeleteInTx keeps instructor role for non-approved rows and propagates revert failure", func(t *testing.T) {
		rejectedState := &mockDBState{onQuery: appAndUserRows(entities.ApplicationStatusRejected, entities.RoleInstructor)}
		rejectedRepo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, rejectedState))
		if err := rejectedRepo.DeleteInTx(context.Background(), "app-1"); err != nil {
			t.Fatalf("unexpected error deleting rejected app: %v", err)
		}
		for _, ex := range rejectedState.execs {
			if strings.Contains(ex.sql, "UPDATE `users`") {
				t.Fatalf("deleting rejected app must not demote active instructor: %s", ex.sql)
			}
		}

		failState := &mockDBState{
			onQuery: appAndUserRows(entities.ApplicationStatusApproved, entities.RoleInstructor),
			onExec: func(q string, _ []driver.Value) (execResponse, bool) {
				if strings.Contains(q, "UPDATE `users`") {
					return execResponse{err: errors.New("db write failure")}, true
				}
				return execResponse{}, false
			},
		}
		failRepo := NewMySQLTeacherApplicationRepository(newLockingTestGormDB(t, failState))
		err := failRepo.DeleteInTx(context.Background(), "app-1")
		if err == nil || !strings.Contains(err.Error(), "db write failure") {
			t.Fatalf("expected role revert error propagation, got %v", err)
		}
		for _, ex := range failState.execs {
			if strings.Contains(ex.sql, "DELETE FROM `teacher_applications`") {
				t.Fatalf("delete must not execute after failed role revert: %s", ex.sql)
			}
		}
		if !hasPrefixEvent(failState.events, "ROLLBACK") {
			t.Fatalf("expected ROLLBACK, events=%v", failState.events)
		}
	})
}
