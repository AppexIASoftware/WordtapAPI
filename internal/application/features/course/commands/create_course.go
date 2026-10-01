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

type CreateCourseCommand struct {
	Title         string              `json:"title"`
	Description   *string             `json:"description"`
	Level         string              `json:"level"`
	AccessTier    entities.AccessTier `json:"access_tier"`
	SourceLang    string              `json:"source_lang"`
	TargetLang    string              `json:"target_lang"`
	CoverImageURL *string             `json:"cover_image_url"`
	AuthorID      string              `json:"-"`
}

type CreateCourseHandler struct {
	courseRepo repositories.CourseRepository
}

func NewCreateCourseHandler(courseRepo repositories.CourseRepository) *CreateCourseHandler {
	return &CreateCourseHandler{courseRepo: courseRepo}
}

func (h *CreateCourseHandler) Handle(ctx context.Context, cmd CreateCourseCommand) (*entities.Course, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New("course title is required")
	}
	if cmd.AuthorID == "" {
		return nil, errors.New("author id is required")
	}

	slug := slugify(title)

	level := strings.TrimSpace(cmd.Level)
	if level == "" {
		level = "beginner"
	}

	accessTier := cmd.AccessTier
	if accessTier == "" {
		accessTier = entities.AccessTierFree
	}

	sourceLang := strings.TrimSpace(cmd.SourceLang)
	if sourceLang == "" {
		sourceLang = "es"
	}

	targetLang := strings.TrimSpace(cmd.TargetLang)
	if targetLang == "" {
		targetLang = "en"
	}

	course := &entities.Course{
		ID:            uuid.NewString(),
		Title:         title,
		Slug:          slug,
		Description:   cmd.Description,
		AccessTier:    accessTier,
		Level:         level,
		Status:        entities.ContentStatusDraft,
		SortOrder:     0,
		CoverImageURL: cmd.CoverImageURL,
		SourceLang:    sourceLang,
		TargetLang:    targetLang,
		CreatedBy:     &cmd.AuthorID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.courseRepo.Create(ctx, course); err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}

	return course, nil
}

func slugify(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	clean := nonAlphanumericRegex.ReplaceAllString(lower, "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "course"
	}

	// Suffix 4 random bytes to guarantee uniqueness
	suffix := make([]byte, 2)
	_, _ = rand.Read(suffix)
	return fmt.Sprintf("%s-%s", clean, hex.EncodeToString(suffix))
}
