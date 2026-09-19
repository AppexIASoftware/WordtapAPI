package mysql

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

/**
 * Siembra los registros iniciales esenciales de forma idempotente.
 */
func Seed(db *gorm.DB) error {
	fmt.Println("Seeding initial database records...")

	// 1. Categorías de contenido
	categories := []entities.ContentCategory{
		{
			Name:        "Vocabulario básico",
			Slug:        "vocabulario-basico",
			Description: stringPtr("Palabras esenciales para comenzar."),
			IsActive:    true,
		},
		{
			Name:        "Frases comunes",
			Slug:        "frases-comunes",
			Description: stringPtr("Expresiones utilizadas en conversaciones cotidianas."),
			IsActive:    true,
		},
		{
			Name:        "Viajes",
			Slug:        "viajes",
			Description: stringPtr("Inglés práctico para viajar."),
			IsActive:    true,
		},
		{
			Name:        "Trabajo",
			Slug:        "trabajo",
			Description: stringPtr("Vocabulario útil para contextos laborales."),
			IsActive:    true,
		},
	}

	for _, cat := range categories {
		var existing entities.ContentCategory
		if err := db.Where("slug = ?", cat.Slug).First(&existing).Error; err != nil {
			if err := db.Create(&cat).Error; err != nil {
				return fmt.Errorf("failed to seed category %s: %w", cat.Slug, err)
			}
		}
	}

	// 2. Cursos
	courses := []entities.Course{
		{
			Title:       "Inglés básico gratuito",
			Slug:        "ingles-basico-gratuito",
			Description: stringPtr("Fundamentos de vocabulario y frases."),
			AccessTier:  entities.AccessTierFree,
			Level:       "beginner",
			Status:      entities.ContentStatusPublished,
			SortOrder:   1,
		},
		{
			Title:       "Curso premium WordTap",
			Slug:        "curso-premium-wordtap",
			Description: stringPtr("Contenido avanzado y ejercicios exclusivos."),
			AccessTier:  entities.AccessTierCourse,
			Level:       "intermediate",
			Status:      entities.ContentStatusPublished,
			SortOrder:   2,
		},
	}

	for _, c := range courses {
		var existing entities.Course
		if err := db.Where("slug = ?", c.Slug).First(&existing).Error; err != nil {
			if err := db.Create(&c).Error; err != nil {
				return fmt.Errorf("failed to seed course %s: %w", c.Slug, err)
			}
		}
	}

	// 3. Productos
	products := []entities.Product{
		{
			Name:        "Curso premium WordTap",
			Description: stringPtr("Acceso permanente al contenido premium."),
			ProductType: entities.ProductTypeOneTimeCourse,
			Provider:    "stripe",
			PriceCents:  1999,
			Currency:    "USD",
			AccessTier:  entities.AccessTierCourse,
			IsActive:    true,
		},
		{
			Name:        "Suscripción mensual WordTap",
			Description: stringPtr("Acceso mensual al contenido premium."),
			ProductType: entities.ProductTypeMonthlySubscription,
			Provider:    "stripe",
			PriceCents:  999,
			Currency:    "USD",
			AccessTier:  entities.AccessTierSubscription,
			IsActive:    true,
		},
	}

	for _, p := range products {
		var count int64
		db.Model(&entities.Product{}).Where("name = ? AND product_type = ?", p.Name, p.ProductType).Count(&count)
		if count == 0 {
			if err := db.Create(&p).Error; err != nil {
				return fmt.Errorf("failed to seed product %s: %w", p.Name, err)
			}
		}
	}

	// 4. Elementos de vocabulario
	var defaultCategory entities.ContentCategory
	db.Where("slug = ?", "vocabulario-basico").First(&defaultCategory)
	var catID *string
	if defaultCategory.ID != "" {
		catID = &defaultCategory.ID
	}

	vocabList := []entities.VocabularyItem{
		{Spanish: "Gato", English: "Cat", Definition: stringPtr("Animal doméstico."), Status: entities.ContentStatusPublished, CategoryID: catID},
		{Spanish: "Perro", English: "Dog", Definition: stringPtr("Animal doméstico."), Status: entities.ContentStatusPublished, CategoryID: catID},
		{Spanish: "Casa", English: "House", Definition: stringPtr("Lugar donde vive una persona."), Status: entities.ContentStatusPublished, CategoryID: catID},
		{Spanish: "Libro", English: "Book", Definition: stringPtr("Conjunto de páginas para leer."), Status: entities.ContentStatusPublished, CategoryID: catID},
		{Spanish: "Agua", English: "Water", Definition: stringPtr("Líquido esencial para la vida."), Status: entities.ContentStatusPublished, CategoryID: catID},
		{Spanish: "Comida", English: "Food", Definition: stringPtr("Alimento para consumir."), Status: entities.ContentStatusPublished, CategoryID: catID},
	}

	for _, v := range vocabList {
		var count int64
		db.Model(&entities.VocabularyItem{}).Where("spanish = ? AND english = ?", v.Spanish, v.English).Count(&count)
		if count == 0 {
			if err := db.Create(&v).Error; err != nil {
				return fmt.Errorf("failed to seed vocab %s: %w", v.English, err)
			}
		}
	}

	fmt.Println("Initial database seed completed successfully!")
	return nil
}

func stringPtr(s string) *string {
	return &s
}
