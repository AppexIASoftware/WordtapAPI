package commands

import (
	"context"

	gamecommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	lessonaccess "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type CreateLessonGameCommand struct {
	LessonID      string
	RequesterID   string
	RequesterRole entities.UserRole
	Game          gamecommands.CreateTeacherGameCommand
}

type CreateLessonGameHandler struct {
	lessons repositories.LessonRepository
	courses repositories.CourseRepository
	create  *gamecommands.CreateTeacherGameHandler
}

func NewCreateLessonGameHandler(lessons repositories.LessonRepository, courses repositories.CourseRepository, create *gamecommands.CreateTeacherGameHandler) *CreateLessonGameHandler {
	return &CreateLessonGameHandler{lessons: lessons, courses: courses, create: create}
}

func (h *CreateLessonGameHandler) Handle(ctx context.Context, cmd CreateLessonGameCommand) (*entities.TeacherGame, error) {
	lesson, course, err := loadLessonCourse(ctx, h.lessons, h.courses, cmd.LessonID)
	if err != nil {
		return nil, err
	}
	if !lessonaccess.CanLinkGameToCourse(course, cmd.RequesterID, cmd.RequesterRole) {
		return nil, ErrLessonGameAccessDenied
	}
	game := cmd.Game
	game.TeacherID = cmd.RequesterID
	game.OwnerType = course.OwnerType
	game.CourseID = &lesson.CourseID
	game.Level = course.Level
	game.SourceType = "course_lesson"
	game.SourceID = &lesson.ID
	game.CustomWordsJSON = nil
	return h.create.Handle(ctx, game)
}
