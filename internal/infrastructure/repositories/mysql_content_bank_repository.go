package repositories

import (
	"context"
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

func (r *MySQLContentBankRepository) ListItems(ctx context.Context, filter repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&entities.ContentBankItem{})

	if filter.OnlyMine && filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	} else if filter.CreatedBy == nil {
		query = query.Where("created_by IS NULL")
	}

	if filter.BankType != nil && *filter.BankType != "" && *filter.BankType != "all" {
		query = query.Where("bank_type = ?", *filter.BankType)
	}

	if filter.CEFRLevel != nil && *filter.CEFRLevel != "" && *filter.CEFRLevel != "all" {
		query = query.Where("cefr_level = ?", *filter.CEFRLevel)
	}

	if filter.Difficulty != nil && *filter.Difficulty > 0 {
		query = query.Where("difficulty = ?", *filter.Difficulty)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.Search != nil {
		term := strings.TrimSpace(*filter.Search)
		if term != "" {
			pattern := "%" + strings.ToLower(term) + "%"
			query = query.Where("LOWER(prompt_text) LIKE ? OR LOWER(target_text) LIKE ? OR LOWER(COALESCE(ipa, '')) LIKE ?", pattern, pattern, pattern)
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
		Order("created_at DESC").
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

func (r *MySQLContentBankRepository) UpdateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return r.db.WithContext(ctx).Omit("BankTypeRef").Save(item).Error
}

func (r *MySQLContentBankRepository) DeleteItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.ContentBankItem{}).Error
}
