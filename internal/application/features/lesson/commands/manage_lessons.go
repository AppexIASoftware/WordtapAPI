package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyLesson(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	clean := nonAlphanumericRegex.ReplaceAllString(lower, "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "lesson"
	}
	suffix := make([]byte, 2)
	_, _ = rand.Read(suffix)
	return fmt.Sprintf("%s-%s", clean, hex.EncodeToString(suffix))
}

// --- Create Lesson ---

type CreateLessonCommand struct {
	CourseID         string
	Title            string
	Description      *string
	EstimatedMinutes *int
	SortOrder        int
	RequesterID      string
	RequesterRole    entities.UserRole
}

type CreateLessonHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
}

func NewCreateLessonHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
) *CreateLessonHandler {
	return &CreateLessonHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
	}
}

func (h *CreateLessonHandler) Handle(ctx context.Context, cmd CreateLessonCommand) (*entities.Lesson, error) {
	if cmd.CourseID == "" {
		return nil, errors.New("course id is required")
	}
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New("lesson title is required")
	}

	course, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: only course author or admin can create lessons")
		}
	}

	lesson := &entities.Lesson{
		ID:               uuid.NewString(),
		CourseID:         cmd.CourseID,
		Title:            title,
		Slug:             slugifyLesson(title),
		Description:      cmd.Description,
		AccessTier:       course.AccessTier,
		Status:           entities.ContentStatusDraft,
		SortOrder:        cmd.SortOrder,
		EstimatedMinutes: cmd.EstimatedMinutes,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.lessonRepo.Create(ctx, lesson); err != nil {
		return nil, fmt.Errorf("failed to create lesson: %w", err)
	}

	return lesson, nil
}

// --- Update Lesson ---

type UpdateLessonCommand struct {
	LessonID         string
	Title            *string
	Description      *string
	EstimatedMinutes *int
	SortOrder        *int
	RequesterID      string
	RequesterRole    entities.UserRole
}

type UpdateLessonHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
}

func NewUpdateLessonHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
) *UpdateLessonHandler {
	return &UpdateLessonHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
	}
}

func (h *UpdateLessonHandler) Handle(ctx context.Context, cmd UpdateLessonCommand) (*entities.Lesson, error) {
	lesson, err := h.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return nil, fmt.Errorf("lesson not found: %w", err)
	}

	course, err := h.courseRepo.FindByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, fmt.Errorf("associated course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: only course author or admin can update this lesson")
		}
	}

	if cmd.Title != nil {
		trimmed := strings.TrimSpace(*cmd.Title)
		if trimmed != "" {
			lesson.Title = trimmed
		}
	}
	if cmd.Description != nil {
		lesson.Description = cmd.Description
	}
	if cmd.EstimatedMinutes != nil {
		lesson.EstimatedMinutes = cmd.EstimatedMinutes
	}
	if cmd.SortOrder != nil {
		lesson.SortOrder = *cmd.SortOrder
	}
	lesson.UpdatedAt = time.Now()

	if err := h.lessonRepo.Update(ctx, lesson); err != nil {
		return nil, fmt.Errorf("failed to update lesson: %w", err)
	}

	return lesson, nil
}

// --- Delete Lesson ---

type DeleteLessonCommand struct {
	LessonID      string
	RequesterID   string
	RequesterRole entities.UserRole
}

type DeleteLessonHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
}

func NewDeleteLessonHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
) *DeleteLessonHandler {
	return &DeleteLessonHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
	}
}

func (h *DeleteLessonHandler) Handle(ctx context.Context, cmd DeleteLessonCommand) error {
	lesson, err := h.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return fmt.Errorf("lesson not found: %w", err)
	}

	course, err := h.courseRepo.FindByID(ctx, lesson.CourseID)
	if err != nil {
		return fmt.Errorf("associated course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return errors.New("forbidden: only course author or admin can delete this lesson")
		}
	}

	return h.lessonRepo.Delete(ctx, cmd.LessonID)
}

// --- Save Lesson Item (Card/Step) ---

type SaveLessonItemCommand struct {
	ItemID        string
	LessonID      string
	ItemType      entities.LessonContentType
	ContentText   *string
	SortOrder     int
	IsRequired    bool
	RequesterID   string
	RequesterRole entities.UserRole
}

type SaveLessonItemHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
}

func NewSaveLessonItemHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
) *SaveLessonItemHandler {
	return &SaveLessonItemHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
	}
}

func (h *SaveLessonItemHandler) Handle(ctx context.Context, cmd SaveLessonItemCommand) (*entities.LessonItem, error) {
	lesson, err := h.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return nil, fmt.Errorf("lesson not found: %w", err)
	}

	course, err := h.courseRepo.FindByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, fmt.Errorf("associated course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: only course author or admin can modify lesson cards")
		}
	}

	itemID := cmd.ItemID
	if itemID == "" {
		itemID = uuid.NewString()
	}

	itemType := cmd.ItemType
	if itemType == "" {
		itemType = entities.LessonContentTypeWord
	}

	item := &entities.LessonItem{
		ID:          itemID,
		LessonID:    cmd.LessonID,
		ItemType:    itemType,
		ContentText: cmd.ContentText,
		SortOrder:   cmd.SortOrder,
		IsRequired:  cmd.IsRequired,
	}

	if err := h.lessonRepo.SaveItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to save lesson card: %w", err)
	}

	return item, nil
}

// --- Delete Lesson Item ---

type DeleteLessonItemCommand struct {
	LessonID      string
	ItemID        string
	RequesterID   string
	RequesterRole entities.UserRole
}

type DeleteLessonItemHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
}

func NewDeleteLessonItemHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
) *DeleteLessonItemHandler {
	return &DeleteLessonItemHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
	}
}

func (h *DeleteLessonItemHandler) Handle(ctx context.Context, cmd DeleteLessonItemCommand) error {
	lesson, err := h.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return fmt.Errorf("lesson not found: %w", err)
	}

	course, err := h.courseRepo.FindByID(ctx, lesson.CourseID)
	if err != nil {
		return fmt.Errorf("associated course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return errors.New("forbidden: only course author or admin can delete lesson cards")
		}
	}

	return h.lessonRepo.DeleteItem(ctx, cmd.ItemID)
}
