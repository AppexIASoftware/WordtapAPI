package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

func lockOrInitLearningStats(tx *gorm.DB, userID string) (*entities.UserLearningStats, error) {
	seed := entities.UserLearningStats{
		UserID:         userID,
		LivesRemaining: 5,
		MaxLives:       5,
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return nil, err
	}
	var stats entities.UserLearningStats
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&stats).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *MySQLUserRepository) GetLearningStats(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	var stats entities.UserLearningStats
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&stats).Error
	if err == nil {
		return &stats, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var initialized *entities.UserLearningStats
	if txErr := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockErr error
		initialized, lockErr = lockOrInitLearningStats(tx, userID)
		return lockErr
	}); txErr != nil {
		return nil, txErr
	}
	return initialized, nil
}

func (r *MySQLUserRepository) DeductUserLife(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	var stats *entities.UserLearningStats
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		stats, txErr = lockOrInitLearningStats(tx, userID)
		if txErr != nil {
			return txErr
		}
		if !stats.DeductLife() {
			return nil
		}
		now := time.Now().UTC()
		stats.UpdatedAt = now
		return tx.Model(&entities.UserLearningStats{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"lives_remaining":   stats.LivesRemaining,
				"last_life_lost_at": stats.LastLifeLostAt,
				"updated_at":        now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *MySQLUserRepository) RefillUserLives(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	var stats *entities.UserLearningStats
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		stats, txErr = lockOrInitLearningStats(tx, userID)
		if txErr != nil {
			return txErr
		}
		stats.RefillLives()
		now := time.Now().UTC()
		stats.UpdatedAt = now
		return tx.Model(&entities.UserLearningStats{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"lives_remaining":   stats.LivesRemaining,
				"last_life_lost_at": nil,
				"updated_at":        now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *MySQLUserRepository) RecordGameVictory(ctx context.Context, params domainRepo.RecordProgressParams) (*domainRepo.RecordProgressResult, error) {
	now := time.Now().UTC()
	res := &domainRepo.RecordProgressResult{
		StarsEarned: params.StarsEarned,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stats, err := lockOrInitLearningStats(tx, params.UserID)
		if err != nil {
			return err
		}

		if params.LessonID != "" {
			seedProg := entities.UserLessonProgress{
				UserID:         params.UserID,
				LessonID:       params.LessonID,
				CompletedItems: 0,
				TotalItems:     params.WordsPracticed,
				StartedAt:      &now,
				UpdatedAt:      now,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seedProg).Error; err != nil {
				return err
			}
			var prog entities.UserLessonProgress
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND lesson_id = ?", params.UserID, params.LessonID).
				First(&prog).Error; err != nil {
				return err
			}
			wasCompleted := prog.CompletedAt != nil || prog.ProgressPercent >= 100.0
			if params.WordsPracticed > prog.TotalItems {
				prog.TotalItems = params.WordsPracticed
			}
			if prog.TotalItems <= 0 {
				prog.TotalItems = 1
			}
			prog.CompletedItems = prog.TotalItems
			prog.ProgressPercent = 100.0
			prog.CompletedAt = &now
			prog.UpdatedAt = now
			if err := tx.Save(&prog).Error; err != nil {
				return err
			}
			if !wasCompleted {
				stats.LessonsCompleted++
			}
			res.LessonCompleted = true
		}

		stats.RecordVictory(params.StarsEarned, params.WordsPracticed, now)
		if err := tx.Save(stats).Error; err != nil {
			return err
		}

		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		seedDaily := entities.UserDailyActivity{
			UserID:       params.UserID,
			ActivityDate: today,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seedDaily).Error; err != nil {
			return err
		}
		var daily entities.UserDailyActivity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND activity_date = ?", params.UserID, today).
			First(&daily).Error; err != nil {
			return err
		}
		points := params.StarsEarned * 10
		if points == 0 {
			points = 5
		}
		daily.MinutesStudied += 2
		daily.QuestionsAnswered += params.WordsPracticed
		daily.CorrectAnswers += params.WordsPracticed
		daily.PointsEarned += points
		if err := tx.Save(&daily).Error; err != nil {
			return err
		}

		res.TotalStars = stats.TotalStars
		res.TotalPoints = stats.TotalPoints
		res.CurrentStreakDays = stats.CurrentStreakDays
		res.WordsLearned = stats.WordsLearned
		res.LessonsCompleted = stats.LessonsCompleted
		res.GamesCompleted = stats.GamesCompleted

		return nil
	})

	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *MySQLUserRepository) GetCompletedLessonIDs(ctx context.Context, userID string) ([]string, error) {
	var lessonIDs []string
	err := r.db.WithContext(ctx).
		Model(&entities.UserLessonProgress{}).
		Where("user_id = ? AND (completed_at IS NOT NULL OR progress_percent >= 100.0)", userID).
		Pluck("lesson_id", &lessonIDs).Error
	if err != nil {
		return nil, err
	}
	return lessonIDs, nil
}
