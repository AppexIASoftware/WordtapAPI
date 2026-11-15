package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ErrStudentIDRequired is returned when student ID is empty.
var ErrStudentIDRequired = errors.New("student id is required")

// UpdateStudentStatusCommand toggles a student's active status.
type UpdateStudentStatusCommand struct {
	StudentID string
	IsActive  bool
}

// UpdateStudentStatusHandler handles suspending or reactivating a student account.
type UpdateStudentStatusHandler struct {
	repo repositories.AdminStudentRepository
}

// NewUpdateStudentStatusHandler creates an UpdateStudentStatusHandler.
func NewUpdateStudentStatusHandler(repo repositories.AdminStudentRepository) *UpdateStudentStatusHandler {
	return &UpdateStudentStatusHandler{repo: repo}
}

// Handle updates the student's active flag and revokes sessions when suspended.
func (h *UpdateStudentStatusHandler) Handle(ctx context.Context, cmd UpdateStudentStatusCommand) (*repositories.AdminStudentStatusResult, error) {
	id := strings.TrimSpace(cmd.StudentID)
	if id == "" {
		return nil, ErrStudentIDRequired
	}
	return h.repo.UpdateStudentStatus(ctx, id, cmd.IsActive)
}
