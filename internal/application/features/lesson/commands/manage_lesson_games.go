package commands

import (
	"context"
	"fmt"
	"strings"

	lessonAccess "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrLessonGameLessonNotFound = fmt.Errorf("%w", apperrors.New("No se encontró la lección."))
	ErrLessonGameGameNotFound   = fmt.Errorf("%w", apperrors.New("No se encontró la actividad."))
	ErrLessonGameLinkNotFound   = fmt.Errorf("%w", apperrors.New("No se encontró el vínculo."))
	ErrLessonGameAccessDenied   = fmt.Errorf("%w", apperrors.New("No tienes permisos para modificar los juegos de esta lección."))
	ErrLessonGameDuplicate      = fmt.Errorf("%w", apperrors.New("La actividad ya está vinculada a esta lección."))
	ErrLessonGameLevelTooHigh   = fmt.Errorf("%w", apperrors.New("El nivel de la actividad supera el nivel del curso."))
)

// loadLessonCourse loads lesson and course or returns not found error.
func loadLessonCourse(
	ctx context.Context,
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
	lessonID string,
) (*entities.Lesson, *entities.Course, error) {
	if strings.TrimSpace(lessonID) == "" {
		return nil, nil, ErrLessonGameLessonNotFound
	}

	lesson, err := lessonRepo.FindByID(ctx, lessonID)
	if err != nil || lesson == nil {
		return nil, nil, ErrLessonGameLessonNotFound
	}

	course, err := courseRepo.FindByID(ctx, lesson.CourseID)
	if err != nil || course == nil {
		return nil, nil, ErrLessonGameLessonNotFound
	}

	return lesson, course, nil
}

// isGameAccessible checks active state, explicit global scope or authorship.
func isGameAccessible(game *entities.TeacherGame, requesterID string) bool {
	if game == nil || !game.IsActive {
		return false
	}
	if game.IsGlobal {
		return true
	}
	return game.TeacherID == requesterID
}

// --- Link Game ---

type LinkLessonGameCommand struct {
	LessonID      string
	GameID        string
	SortOrder     *int
	IsRequired    *bool
	RequesterID   string
	RequesterRole entities.UserRole
}

type LinkLessonGameHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
	gameRepo   repositories.TeacherGameRepository
	linkRepo   repositories.LessonGameRepository
}

func NewLinkLessonGameHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
	gameRepo repositories.TeacherGameRepository,
	linkRepo repositories.LessonGameRepository,
) *LinkLessonGameHandler {
	return &LinkLessonGameHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
		gameRepo:   gameRepo,
		linkRepo:   linkRepo,
	}
}

func (h *LinkLessonGameHandler) Handle(ctx context.Context, cmd LinkLessonGameCommand) (*entities.LessonGame, error) {
	if strings.TrimSpace(cmd.GameID) == "" {
		return nil, apperrors.New("Falta el identificador de la actividad.")
	}

	_, course, err := loadLessonCourse(ctx, h.lessonRepo, h.courseRepo, cmd.LessonID)
	if err != nil {
		return nil, err
	}
	if !lessonAccess.CanLinkGameToCourse(course, cmd.RequesterID, cmd.RequesterRole) {
		return nil, ErrLessonGameAccessDenied
	}

	game, err := h.gameRepo.FindByID(ctx, cmd.GameID)
	if err != nil || game == nil || !isGameAccessible(game, cmd.RequesterID) {
		return nil, ErrLessonGameGameNotFound
	}
	// A source lesson is not transferable, even when old data also marks it global.
	if game.SourceType == "course_lesson" && game.SourceID != nil && strings.TrimSpace(*game.SourceID) != "" {
		if game.IsGlobal || *game.SourceID != cmd.LessonID {
			return nil, ErrLessonGameGameNotFound
		}
	}

	levelCompat := entities.CompareLevels(course.Level, game.Level)
	if levelCompat == entities.LevelHigher {
		return nil, ErrLessonGameLevelTooHigh
	}

	existing, err := h.linkRepo.FindByLessonAndGame(ctx, cmd.LessonID, cmd.GameID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrLessonGameDuplicate
	}

	sortOrder := 0
	if cmd.SortOrder != nil {
		if *cmd.SortOrder < 0 {
			return nil, apperrors.New("El orden no puede ser negativo.")
		}
		sortOrder = *cmd.SortOrder
	} else {
		sortOrder, err = h.linkRepo.NextSortOrder(ctx, cmd.LessonID)
		if err != nil {
			return nil, err
		}
	}

	isRequired := true
	if cmd.IsRequired != nil {
		isRequired = *cmd.IsRequired
	}

	link := &entities.LessonGame{
		LessonID:      cmd.LessonID,
		TeacherGameID: cmd.GameID,
		SortOrder:     sortOrder,
		IsRequired:    isRequired,
		TeacherGame:   game,
		LevelWarning:  entities.LevelWarningMessage(levelCompat),
	}
	if err := h.linkRepo.Create(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}

// --- Update Link ---

type UpdateLessonGameCommand struct {
	LessonID      string
	GameID        string
	SortOrder     *int
	IsRequired    *bool
	RequesterID   string
	RequesterRole entities.UserRole
}

type UpdateLessonGameHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
	linkRepo   repositories.LessonGameRepository
}

func NewUpdateLessonGameHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
	linkRepo repositories.LessonGameRepository,
) *UpdateLessonGameHandler {
	return &UpdateLessonGameHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
		linkRepo:   linkRepo,
	}
}

func (h *UpdateLessonGameHandler) Handle(ctx context.Context, cmd UpdateLessonGameCommand) (*entities.LessonGame, error) {
	_, course, err := loadLessonCourse(ctx, h.lessonRepo, h.courseRepo, cmd.LessonID)
	if err != nil {
		return nil, err
	}
	if !lessonAccess.CanLinkGameToCourse(course, cmd.RequesterID, cmd.RequesterRole) {
		return nil, ErrLessonGameAccessDenied
	}

	link, err := h.linkRepo.FindByLessonAndGame(ctx, cmd.LessonID, cmd.GameID)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, ErrLessonGameLinkNotFound
	}

	if cmd.SortOrder != nil {
		if *cmd.SortOrder < 0 {
			return nil, apperrors.New("El orden no puede ser negativo.")
		}
		link.SortOrder = *cmd.SortOrder
	}
	if cmd.IsRequired != nil {
		link.IsRequired = *cmd.IsRequired
	}

	if err := h.linkRepo.Update(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

// --- Unlink Game ---

type UnlinkLessonGameCommand struct {
	LessonID      string
	GameID        string
	RequesterID   string
	RequesterRole entities.UserRole
}

type UnlinkLessonGameHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
	linkRepo   repositories.LessonGameRepository
}

func NewUnlinkLessonGameHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
	linkRepo repositories.LessonGameRepository,
) *UnlinkLessonGameHandler {
	return &UnlinkLessonGameHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
		linkRepo:   linkRepo,
	}
}

func (h *UnlinkLessonGameHandler) Handle(ctx context.Context, cmd UnlinkLessonGameCommand) error {
	_, course, err := loadLessonCourse(ctx, h.lessonRepo, h.courseRepo, cmd.LessonID)
	if err != nil {
		return err
	}
	if !lessonAccess.CanLinkGameToCourse(course, cmd.RequesterID, cmd.RequesterRole) {
		return ErrLessonGameAccessDenied
	}

	link, err := h.linkRepo.FindByLessonAndGame(ctx, cmd.LessonID, cmd.GameID)
	if err != nil {
		return err
	}
	if link == nil {
		return nil
	}

	return h.linkRepo.Delete(ctx, cmd.LessonID, cmd.GameID)
}
