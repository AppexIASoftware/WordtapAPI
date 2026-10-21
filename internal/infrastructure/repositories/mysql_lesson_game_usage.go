package repositories

import (
	"context"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// lessonGameUsageRow projects grouped usage count by game.
type lessonGameUsageRow struct {
	TeacherGameID string `gorm:"column:teacher_game_id"`
	UsageCount    int    `gorm:"column:usage_count"`
}

// countLessonGamesByGameIDs counts lesson links for each game ID.
func countLessonGamesByGameIDs(ctx context.Context, db *gorm.DB, ids []string) (map[string]int, error) {
	counts := make(map[string]int, len(ids))
	if len(ids) == 0 {
		return counts, nil
	}

	rows := make([]lessonGameUsageRow, 0, len(ids))
	err := db.WithContext(ctx).
		Model(&entities.LessonGame{}).
		Select("teacher_game_id, COUNT(*) AS usage_count").
		Where("teacher_game_id IN ?", ids).
		Group("teacher_game_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.TeacherGameID] = row.UsageCount
	}
	return counts, nil
}

// countLessonGamesByGameID counts lesson links for a single game.
func countLessonGamesByGameID(ctx context.Context, db *gorm.DB, gameID string) (int, error) {
	counts, err := countLessonGamesByGameIDs(ctx, db, []string{gameID})
	if err != nil {
		return 0, err
	}
	return counts[gameID], nil
}
