package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type MySQLContentBankRepository struct {
	db *gorm.DB
}

func NewMySQLContentBankRepository(db *gorm.DB) *MySQLContentBankRepository {
	return &MySQLContentBankRepository{db: db}
}

func (r *MySQLContentBankRepository) ListTypes(ctx context.Context) ([]entities.ContentBankType, error) {
	var types []entities.ContentBankType
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("display_name ASC").
		Find(&types).Error
	if err != nil {
		return nil, err
	}
	return types, nil
}

func (r *MySQLContentBankRepository) FindTypeBySlug(ctx context.Context, slug string) (*entities.ContentBankType, error) {
	var bankType entities.ContentBankType
	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&bankType).Error
	if err != nil {
		return nil, err
	}
	return &bankType, nil
}

func (r *MySQLContentBankRepository) ListCategories(ctx context.Context) ([]entities.ContentCategory, error) {
	var categories []entities.ContentCategory
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *MySQLContentBankRepository) CreateCategory(ctx context.Context, category *entities.ContentCategory) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *MySQLContentBankRepository) ListItems(ctx context.Context, filter repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&entities.ContentBankItem{})

	if filter.OnlyMine && filter.CreatedBy != nil {
		query = query.Where("content_bank_items.created_by = ?", *filter.CreatedBy)
	} else if filter.CreatedBy == nil {
		query = query.Where("content_bank_items.created_by IS NULL")
	}

	if filter.BankType != nil && *filter.BankType != "" && *filter.BankType != "all" {
		query = query.Where("content_bank_items.bank_type = ?", *filter.BankType)
	}

	if filter.CategoryID != nil && *filter.CategoryID != "" && *filter.CategoryID != "all" {
		catVal := strings.TrimSpace(*filter.CategoryID)
		query = query.Joins("LEFT JOIN content_categories ON content_categories.id = content_bank_items.category_id").
			Where("content_bank_items.category_id = ? OR content_categories.slug = ?", catVal, catVal)
	}

	if filter.CEFRLevel != nil && *filter.CEFRLevel != "" && *filter.CEFRLevel != "all" {
		query = query.Where("content_bank_items.cefr_level = ?", *filter.CEFRLevel)
	}

	if filter.Difficulty != nil && *filter.Difficulty > 0 {
		query = query.Where("content_bank_items.difficulty = ?", *filter.Difficulty)
	}

	if filter.Status != nil {
		query = query.Where("content_bank_items.status = ?", *filter.Status)
	}

	if filter.Search != nil {
		term := strings.TrimSpace(*filter.Search)
		if term != "" {
			pattern := "%" + strings.ToLower(term) + "%"
			query = query.Where("LOWER(content_bank_items.prompt_text) LIKE ? OR LOWER(content_bank_items.target_text) LIKE ? OR LOWER(COALESCE(content_bank_items.ipa, '')) LIKE ? OR LOWER(content_bank_items.metadata) LIKE ?", pattern, pattern, pattern, pattern)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var items []entities.ContentBankItem
	err := query.
		Preload("BankTypeRef").
		Preload("Category").
		Order("content_bank_items.created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *MySQLContentBankRepository) FindItemByID(ctx context.Context, id string) (*entities.ContentBankItem, error) {
	var item entities.ContentBankItem
	err := r.db.WithContext(ctx).
		Preload("BankTypeRef").
		Preload("Category").
		Where("id = ?", id).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *MySQLContentBankRepository) CreateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// withReviewGuard protects existing lesson references within the write transaction.
// Link creation and review transitions must participate in the same locking
// protocol before this can protect against concurrent changes to the link set.
func (r *MySQLContentBankRepository) withReviewGuard(ctx context.Context, id string, write func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var courses []struct { Status entities.ContentStatus }
		if err := tx.Raw(`SELECT courses.status FROM courses
			JOIN lessons ON lessons.course_id = courses.id
			JOIN lesson_items ON lesson_items.lesson_id = lessons.id
			WHERE lesson_items.content_bank_item_id = ? ORDER BY courses.id FOR UPDATE`, id).Scan(&courses).Error; err != nil {
			return err
		}
		for _, course := range courses {
			if course.Status == entities.ContentStatusInReview {
				return errors.New("no se puede modificar un ítem usado en un curso en revisión")
			}
		}
		return write(tx)
	})
}

func (r *MySQLContentBankRepository) UpdateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return r.withReviewGuard(ctx, item.ID, func(tx *gorm.DB) error {
		return tx.Omit("BankTypeRef").Save(item).Error
	})
}

func (r *MySQLContentBankRepository) DeleteItem(ctx context.Context, id string) error {
	return r.withReviewGuard(ctx, id, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&entities.ContentBankItem{}).Error
	})
}
