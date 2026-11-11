package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func (r *MySQLUserRepository) UpdateUser(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *MySQLUserRepository) ListUserIDs(ctx context.Context, role *entities.UserRole, learningLevel *string) ([]string, error) {
	var ids []string
	q := r.db.WithContext(ctx).Model(&entities.User{}).Where("is_active = ?", true)
	if role != nil && *role != "" {
		q = q.Where("role = ?", *role)
	}
	if learningLevel != nil && *learningLevel != "" {
		q = q.Where("learning_level = ?", *learningLevel)
	}
	err := q.Pluck("id", &ids).Error
	return ids, err
}
