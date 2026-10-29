package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

const (
	DefaultCourseMinPriceCents = 2000
	CourseMinPriceSettingKey   = "course_min_price_pen"
)

func resolveMinPriceCents(ctx context.Context, repo repositories.PlatformSettingRepository) int {
	if repo == nil {
		return DefaultCourseMinPriceCents
	}
	s, err := repo.Get(ctx, CourseMinPriceSettingKey)
	if err != nil || s == nil {
		return DefaultCourseMinPriceCents
	}
	val, err := strconv.Atoi(strings.TrimSpace(s.Value))
	if err != nil || val <= 0 {
		return DefaultCourseMinPriceCents
	}
	return val
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)

type CreateCourseCommand struct {
	Title         string              `json:"title"`
	Description   *string             `json:"description"`
	Level         string              `json:"level"`
	CategoryID    *string             `json:"category_id"`
	AccessTier    entities.AccessTier `json:"access_tier"`
	PriceCents    int                 `json:"price_cents"`
	SourceLang    string              `json:"source_lang"`
	TargetLang    string              `json:"target_lang"`
	CoverImageURL *string             `json:"cover_image_url"`
	AuthorID      string              `json:"-"`
	OwnerType     entities.OwnerType  `json:"-"`
}

type CreateCourseHandler struct {
	courseRepo  repositories.CourseRepository
	settingRepo repositories.PlatformSettingRepository
}

func NewCreateCourseHandler(courseRepo repositories.CourseRepository, settingRepo ...repositories.PlatformSettingRepository) *CreateCourseHandler {
	var sRepo repositories.PlatformSettingRepository
	if len(settingRepo) > 0 {
		sRepo = settingRepo[0]
	}
	return &CreateCourseHandler{courseRepo: courseRepo, settingRepo: sRepo}
}

func (h *CreateCourseHandler) WithSettingRepo(settingRepo repositories.PlatformSettingRepository) *CreateCourseHandler {
	h.settingRepo = settingRepo
	return h
}

func (h *CreateCourseHandler) Handle(ctx context.Context, cmd CreateCourseCommand) (*entities.Course, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, apperrors.New("El título del curso es obligatorio.")
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

	ownerType := cmd.OwnerType
	if !ownerType.IsValid() {
		ownerType = entities.OwnerTypeTeacher
	}

	priceCents := cmd.PriceCents
	if accessTier == entities.AccessTierSubscription || accessTier == entities.AccessTierFree {
		priceCents = 0
	} else if accessTier == entities.AccessTierCourse {
		if ownerType == entities.OwnerTypeTeacher {
			minPriceCents := resolveMinPriceCents(ctx, h.settingRepo)
			if priceCents < minPriceCents {
				return nil, apperrors.New(fmt.Sprintf("El precio mínimo para cursos individuales de docentes es de S/ %.2f PEN.", float64(minPriceCents)/100.0))
			}
		} else if priceCents == 0 {
			priceCents = resolveMinPriceCents(ctx, h.settingRepo)
		}
	}

	course := &entities.Course{
		ID:            uuid.NewString(),
		Title:         title,
		Slug:          slug,
		Description:   cmd.Description,
		AccessTier:    accessTier,
		PriceCents:    priceCents,
		Level:         level,
		Status:        entities.ContentStatusDraft,
		OwnerType:     ownerType,
		SortOrder:     0,
		CoverImageURL: cmd.CoverImageURL,
		CategoryID:    cmd.CategoryID,
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
