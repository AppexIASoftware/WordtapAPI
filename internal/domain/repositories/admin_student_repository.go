package repositories

import (
	"context"
	"errors"
	"time"
)

// ErrStudentNotFound indicates the requested student does not exist.
var ErrStudentNotFound = errors.New("student not found")

// AdminStudentCourseProgress summarizes a student's progress within a single course.
type AdminStudentCourseProgress struct {
	CourseID         string     `json:"course_id"`
	CourseTitle      string     `json:"course_title"`
	CompletedLessons int        `json:"completed_lessons"`
	TotalLessons     int        `json:"total_lessons"`
	ProgressPercent  int        `json:"progress_percent"`
	LastActivityAt   *time.Time `json:"last_activity_at,omitempty"`
}

// AdminStudentClassroom summarizes a student's enrollment in a classroom.
type AdminStudentClassroom struct {
	ClassroomID   string    `json:"classroom_id"`
	ClassroomName string    `json:"classroom_name"`
	TeacherName   string    `json:"teacher_name"`
	EnrolledAt    time.Time `json:"enrolled_at"`
}

// AdminStudentDetail holds full supervision details for a single student.
type AdminStudentDetail struct {
	ID                string                       `json:"id"`
	Name              string                       `json:"name"`
	Email             string                       `json:"email"`
	LearningLevel     string                       `json:"learning_level"`
	AccessTier        string                       `json:"access_tier"`
	IsActive          bool                         `json:"is_active"`
	CreatedAt         time.Time                    `json:"created_at"`
	CurrentStreakDays int                          `json:"current_streak_days"`
	TotalPoints       int                          `json:"total_points"`
	LessonsCompleted  int                          `json:"lessons_completed"`
	WordsLearned      int                          `json:"words_learned"`
	LastActivityDate  *time.Time                   `json:"last_activity_date,omitempty"`
	Courses           []AdminStudentCourseProgress `json:"courses"`
	Classrooms        []AdminStudentClassroom      `json:"classrooms"`
}

// AdminStudentStatusResult is returned after toggling a student's active status.
type AdminStudentStatusResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}

// AdminStudentRepository abstracts admin supervision queries and mutations on students.
type AdminStudentRepository interface {
	GetStudentDetail(ctx context.Context, studentID string) (*AdminStudentDetail, error)
	UpdateStudentStatus(ctx context.Context, studentID string, isActive bool) (*AdminStudentStatusResult, error)
}
