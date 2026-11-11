package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

func (r *MySQLUserRepository) GetLearningStats(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	var stats entities.UserLearningStats
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&stats).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			stats = entities.UserLearningStats{
				UserID:         userID,
				LivesRemaining: 5,
				MaxLives:       5,
			}
			if createErr := r.db.WithContext(ctx).Create(&stats).Error; createErr != nil {
				return nil, createErr
			}
			return &stats, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (r *MySQLUserRepository) DeductUserLife(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	stats, err := r.GetLearningStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.DeductLife()
	if err := r.db.WithContext(ctx).Model(&entities.UserLearningStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"lives_remaining":   stats.LivesRemaining,
			"last_life_lost_at": stats.LastLifeLostAt,
			"updated_at":        time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *MySQLUserRepository) RefillUserLives(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	stats, err := r.GetLearningStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.RefillLives()
	if err := r.db.WithContext(ctx).Model(&entities.UserLearningStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"lives_remaining":   stats.LivesRemaining,
			"last_life_lost_at": nil,
			"updated_at":        time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *MySQLUserRepository) RecordGameVictory(ctx context.Context, params domainRepo.RecordProgressParams) (*domainRepo.RecordProgressResult, error) {
	now := time.Now()
	res := &domainRepo.RecordProgressResult{
		StarsEarned: params.StarsEarned,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stats entities.UserLearningStats
		if err := tx.Where("user_id = ?", params.UserID).First(&stats).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				stats = entities.UserLearningStats{
					UserID:         params.UserID,
					LivesRemaining: 5,
					MaxLives:       5,
				}
				if err := tx.Create(&stats).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		if params.LessonID != "" {
			var prog entities.UserLessonProgress
			err := tx.Where("user_id = ? AND lesson_id = ?", params.UserID, params.LessonID).First(&prog).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				prog = entities.UserLessonProgress{
					UserID:          params.UserID,
					LessonID:        params.LessonID,
					CompletedItems:  params.WordsPracticed,
					TotalItems:      params.WordsPracticed,
					ProgressPercent: 100.0,
					StartedAt:       &now,
					CompletedAt:     &now,
					UpdatedAt:       now,
				}
				if err := tx.Create(&prog).Error; err != nil {
					return err
				}
				stats.LessonsCompleted++
				res.LessonCompleted = true
			} else {
				wasCompleted := prog.CompletedAt != nil || prog.ProgressPercent >= 100.0
				prog.ProgressPercent = 100.0
				prog.CompletedAt = &now
				if params.WordsPracticed > prog.CompletedItems {
					prog.CompletedItems = params.WordsPracticed
				}
				prog.UpdatedAt = now
				if err := tx.Save(&prog).Error; err != nil {
					return err
				}
				if !wasCompleted {
					stats.LessonsCompleted++
				}
				res.LessonCompleted = true
			}
		}

		stats.RecordVictory(params.StarsEarned, params.WordsPracticed, now)
		if err := tx.Save(&stats).Error; err != nil {
			return err
		}

		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		var daily entities.UserDailyActivity
		dailyErr := tx.Where("user_id = ? AND activity_date = ?", params.UserID, today).First(&daily).Error
		if dailyErr != nil && !errors.Is(dailyErr, gorm.ErrRecordNotFound) {
			return dailyErr
		}
		points := params.StarsEarned * 10
		if points == 0 {
			points = 5
		}
		if errors.Is(dailyErr, gorm.ErrRecordNotFound) {
			daily = entities.UserDailyActivity{
				UserID:            params.UserID,
				ActivityDate:      today,
				MinutesStudied:    2,
				QuestionsAnswered: params.WordsPracticed,
				CorrectAnswers:    params.WordsPracticed,
				PointsEarned:      points,
			}
			if err := tx.Create(&daily).Error; err != nil {
				return err
			}
		} else {
			daily.MinutesStudied += 2
			daily.QuestionsAnswered += params.WordsPracticed
			daily.CorrectAnswers += params.WordsPracticed
			daily.PointsEarned += points
			if err := tx.Save(&daily).Error; err != nil {
				return err
			}
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
