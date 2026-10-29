package gamification

import (
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestGameDTOIncludesUsageAndScope(t *testing.T) {
	game := &entities.TeacherGame{OwnerType: entities.OwnerTypePlatform, IsGlobal: false, UsageCount: 3}
	dto := formatTeacherGameDTO(game)
	if dto["usage_count"] != 3 || dto["is_global"] != false || dto["owner_type"] != entities.OwnerTypePlatform {
		t.Fatalf("incorrect usage or scope: %#v", dto)
	}
}
