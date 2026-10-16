package commands

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type CreateClassroomCommand struct {
	TeacherID       string   `json:"-"`
	Name            string   `json:"name"`
	Level           string   `json:"level"`
	CourseID        *string  `json:"course_id"`
	CourseIDs       []string `json:"course_ids"`
	GameIDs         []string `json:"game_ids"`
	WeeklyGoalWords int      `json:"weekly_goal_words"`
	Description     *string  `json:"description"`
}

type CreateClassroomHandler struct {
	classroomRepo repositories.ClassroomRepository
}

func NewCreateClassroomHandler(classroomRepo repositories.ClassroomRepository) *CreateClassroomHandler {
	return &CreateClassroomHandler{classroomRepo: classroomRepo}
}

func (h *CreateClassroomHandler) Handle(ctx context.Context, cmd CreateClassroomCommand) (*entities.Classroom, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, apperrors.New("El nombre del aula es obligatorio.")
	}
	if cmd.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
	}

	level := strings.TrimSpace(cmd.Level)
	if level == "" {
		level = "B1 Intermediate"
	}

	// Extraer prefijo de nivel (ej. "B1", "A2", "C1")
	levelTag := "B1"
	if len(level) >= 2 {
		tag := strings.ToUpper(level[:2])
		if tag == "A1" || tag == "A2" || tag == "B1" || tag == "B2" || tag == "C1" || tag == "C2" {
			levelTag = tag
		}
	}

	// Generar código alfanumérico único con formato WT-{NIVEL}-{XXXX}
	var uniqueCode string
	for i := 0; i < 5; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(9000))
		if err != nil {
			return nil, fmt.Errorf("error generando código de aula: %w", err)
		}
		candidate := fmt.Sprintf("WT-%s-%04d", levelTag, n.Int64()+1000)

		existing, err := h.classroomRepo.FindByCode(ctx, candidate)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			uniqueCode = candidate
			break
		}
	}

	if uniqueCode == "" {
		return nil, apperrors.New("No fue posible generar un código único de aula. Intenta nuevamente.")
	}

	var assignedCoursesStr *string
	if len(cmd.CourseIDs) > 0 {
		str := strings.Join(cmd.CourseIDs, ",")
		assignedCoursesStr = &str
	} else if cmd.CourseID != nil && *cmd.CourseID != "" {
		assignedCoursesStr = cmd.CourseID
	}

	var activeGamesStr *string
	if len(cmd.GameIDs) > 0 {
		str := strings.Join(cmd.GameIDs, ",")
		activeGamesStr = &str
	} else {
		defaultGames := "match,memory,listening,multiple_choice,speed_translation,word_scramble"
		activeGamesStr = &defaultGames
	}

	weeklyGoal := cmd.WeeklyGoalWords
	if weeklyGoal <= 0 {
		weeklyGoal = 50
	}

	classroom := &entities.Classroom{
		TeacherID:       cmd.TeacherID,
		CourseID:        cmd.CourseID,
		Name:            name,
		Code:            uniqueCode,
		Level:           level,
		Description:     cmd.Description,
		AssignedCourses: assignedCoursesStr,
		ActiveGames:     activeGamesStr,
		WeeklyGoalWords: weeklyGoal,
		IsActive:        true,
	}

	if err := h.classroomRepo.Create(ctx, classroom); err != nil {
		return nil, err
	}

	// Sembrar diagnósticos pedagógicos iniciales de tropiezo según el nivel
	stumbles := defaultStumblesForLevel(classroom.ID, levelTag)
	for i := range stumbles {
		_ = h.classroomRepo.CreateStumble(ctx, &stumbles[i])
	}

	return classroom, nil
}

func defaultStumblesForLevel(classroomID, level string) []entities.StudentStumble {
	switch level {
	case "B2", "C1":
		notes1 := "Confusión fonética recurrente con through en ejercicios rápidos"
		notes2 := "Dificultad ortográfica avanzada nivel B2"
		return []entities.StudentStumble{
			{
				ClassroomID:   classroomID,
				Term:          "thought",
				Translation:   "pensamiento / pasado de think",
				FailRate:      68,
				TotalAttempts: 45,
				GrammarType:   "Pasado irregular",
				Notes:         &notes1,
			},
			{
				ClassroomID:   classroomID,
				Term:          "thorough",
				Translation:   "exhaustivo / minucioso",
				FailRate:      58,
				TotalAttempts: 38,
				GrammarType:   "Adjetivo",
				Notes:         &notes2,
			},
		}
	default:
		notes1 := "Confusión de preposiciones y falsos cognados"
		notes2 := "Confusión con wake up en contrarreloj"
		return []entities.StudentStumble{
			{
				ClassroomID:   classroomID,
				Term:          "through",
				Translation:   "a través de / por",
				FailRate:      62,
				TotalAttempts: 52,
				GrammarType:   "Preposición",
				Notes:         &notes1,
			},
			{
				ClassroomID:   classroomID,
				Term:          "get up",
				Translation:   "levantarse",
				FailRate:      51,
				TotalAttempts: 40,
				GrammarType:   "Phrasal Verb",
				Notes:         &notes2,
			},
		}
	}
}
