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

func (r *MySQLUserRepository) ListStaff(ctx context.Context) ([]entities.User, error) {
	var users []entities.User
	err := r.db.WithContext(ctx).
		Where("role IN ?", []entities.UserRole{
			entities.RoleAdmin,
			entities.RoleModerator,
			entities.RoleInstructor,
		}).
		Order("created_at DESC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *MySQLUserRepository) DeleteUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Delete(&entities.User{}, "id = ?", userID).Error
}
