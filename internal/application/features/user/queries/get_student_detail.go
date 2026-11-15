package queries

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ErrInvalidStudentID is returned when a student ID is missing.
var ErrInvalidStudentID = errors.New("student id is required")

// GetStudentDetailHandler retrieves full supervision details for an admin student view.
type GetStudentDetailHandler struct {
	repo repositories.AdminStudentRepository
}

// NewGetStudentDetailHandler creates a GetStudentDetailHandler.
func NewGetStudentDetailHandler(repo repositories.AdminStudentRepository) *GetStudentDetailHandler {
	return &GetStudentDetailHandler{repo: repo}
}

// Handle fetches detailed progress and enrollments for the specified student.
func (h *GetStudentDetailHandler) Handle(ctx context.Context, studentID string) (*repositories.AdminStudentDetail, error) {
	id := strings.TrimSpace(studentID)
	if id == "" {
		return nil, ErrInvalidStudentID
	}
	return h.repo.GetStudentDetail(ctx, id)
}
