package commands

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrTestAccessDisabled  = errors.New("test course access is disabled")
	ErrInvalidCourseAccess = errors.New("invalid course access request")
)

type TestCourseAccessRepository interface {
	GrantTestAccess(ctx context.Context, userID, courseID string) error
}

type GrantTestCourseAccessHandler struct {
	repository TestCourseAccessRepository
	enabled    bool
}

func NewGrantTestCourseAccessHandler(repository TestCourseAccessRepository, enabled bool) *GrantTestCourseAccessHandler {
	return &GrantTestCourseAccessHandler{repository: repository, enabled: enabled}
}

func (h *GrantTestCourseAccessHandler) Handle(ctx context.Context, userID, courseID string) error {
	if !h.enabled {
		return ErrTestAccessDisabled
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(courseID) == "" || h.repository == nil {
		return ErrInvalidCourseAccess
	}
	return h.repository.GrantTestAccess(ctx, userID, courseID)
}
