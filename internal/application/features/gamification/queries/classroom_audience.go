package queries

import (
	"encoding/json"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func matchesClassroomAudience(game *entities.TeacherGame, classrooms []entities.Classroom) bool {
	if game.AssignedClassroomIDs == nil {
		return false
	}
	var ids []string
	if json.Unmarshal([]byte(*game.AssignedClassroomIDs), &ids) != nil {
		return false
	}
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}
	for _, classroom := range classrooms {
		if classroom.TeacherID == game.TeacherID && classroom.IsActive && allowed[classroom.ID] {
			return true
		}
	}
	return false
}
