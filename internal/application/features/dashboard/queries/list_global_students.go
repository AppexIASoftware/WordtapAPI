package queries

import (
	"context"
	"time"
)

// GlobalStudentFilter holds query parameters for filtering global students.
type GlobalStudentFilter struct {
	Query    string
	Plan     string
	Level    string
	CourseID string
	Status   string
	Limit    int
	Offset   int
}

// GlobalCourseOption represents a published/available course for the filter dropdown.
type GlobalCourseOption struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GlobalStudentItem represents a student row in the global catalog telemetry table.
type GlobalStudentItem struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Email             string     `json:"email"`
	LearningLevel     string     `json:"learning_level"`
	CourseID          string     `json:"course_id"`
	CourseTitle       string     `json:"course_title"`
	CurrentStreakDays int        `json:"current_streak_days"`
	ProgressPercent   int        `json:"progress_percent"`
	AccessTier        string     `json:"access_tier"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	LastActivityDate  *time.Time `json:"last_activity_date,omitempty"`
}

// GlobalStudentsResponse is returned by GET /api/v1/teacher/global-students.
type GlobalStudentsResponse struct {
	Items   []GlobalStudentItem  `json:"items"`
	Total   int64                `json:"total"`
	Courses []GlobalCourseOption `json:"courses"`
}

// GlobalStudentsReader abstracts read-only queries for global students telemetry.
type GlobalStudentsReader interface {
	ListGlobalStudents(ctx context.Context, filter GlobalStudentFilter) (*GlobalStudentsResponse, error)
}

// ListGlobalStudentsHandler handles listing filtered global students.
type ListGlobalStudentsHandler struct {
	reader GlobalStudentsReader
}

// NewListGlobalStudentsHandler creates a ListGlobalStudentsHandler.
func NewListGlobalStudentsHandler(reader GlobalStudentsReader) *ListGlobalStudentsHandler {
	return &ListGlobalStudentsHandler{reader: reader}
}

// Handle executes the global students telemetry query.
func (h *ListGlobalStudentsHandler) Handle(ctx context.Context, filter GlobalStudentFilter) (*GlobalStudentsResponse, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return h.reader.ListGlobalStudents(ctx, filter)
}
