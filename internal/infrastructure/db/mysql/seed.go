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

	// 1.1 Tipos de Bancos de Aprendizaje (Content Vault Types)
	bankTypes := []entities.ContentBankType{
		{Slug: "vocabulary", DisplayName: "Vocabulario", ColorTheme: "emerald", Description: stringPtr("Términos esenciales y partes de la oración"), IsActive: true},
		{Slug: "phrase", DisplayName: "Oraciones", ColorTheme: "blue", Description: stringPtr("Estructuras y patrones sintácticos cotidianos"), IsActive: true},
		{Slug: "collocation", DisplayName: "Colocaciones", ColorTheme: "purple", Description: stringPtr("Frases fijas y combinaciones naturales de palabras"), IsActive: true},
		{Slug: "minimal_pair", DisplayName: "Pares Mínimos", ColorTheme: "amber", Description: stringPtr("Fonética contrastiva crítica para hispanohablantes"), IsActive: true},
		{Slug: "false_friend", DisplayName: "Falsos Amigos", ColorTheme: "rose", Description: stringPtr("Trampas léxicas con traducción contraintuitiva"), IsActive: true},
		{Slug: "dialogue", DisplayName: "Micro-Diálogos", ColorTheme: "cyan", Description: stringPtr("Intercambios conversacionales de 2 o 3 turnos"), IsActive: true},
		{Slug: "idioms", DisplayName: "Modismos & Idioms", ColorTheme: "indigo", Description: stringPtr("Expresiones figuradas no traducibles literalmente"), IsActive: true},
		{Slug: "slang", DisplayName: "Slang & Jerga", ColorTheme: "orange", Description: stringPtr("Lenguaje coloquial y regional auténtico"), IsActive: true},
		{Slug: "business_english", DisplayName: "Inglés de Negocios", ColorTheme: "teal", Description: stringPtr("Términos corporativos y de gestión de proyectos"), IsActive: true},
	}

	for _, bt := range bankTypes {
		var existing entities.ContentBankType
		if err := db.Where("slug = ?", bt.Slug).First(&existing).Error; err != nil {
			if err := db.Create(&bt).Error; err != nil {
				return fmt.Errorf("failed to seed bank type %s: %w", bt.Slug, err)
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

	var basicCourse entities.Course
	db.Where("slug = ?", "ingles-basico-gratuito").First(&basicCourse)
	var courseID string
	if basicCourse.ID != "" {
		courseID = basicCourse.ID
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

	// 5. Lección "Los 50 verbos más usados" y sus elementos iniciales
	if courseID != "" {
		var lesson entities.Lesson
		if err := db.Where("id = ? OR slug = ?", "lesson-1", "los-50-verbos-mas-usados").First(&lesson).Error; err != nil {
			lesson = entities.Lesson{
				ID:               "lesson-1",
				CourseID:         courseID,
				CategoryID:       catID,
				Title:            "Los 50 verbos más usados",
				Slug:             "los-50-verbos-mas-usados",
				Description:      stringPtr("Domina los verbos esenciales del inglés."),
				AccessTier:       entities.AccessTierFree,
				Status:           entities.ContentStatusPublished,
				SortOrder:        1,
				EstimatedMinutes: intPtr(15),
			}
			if err := db.Create(&lesson).Error; err != nil {
				return fmt.Errorf("failed to seed lesson %s: %w", lesson.Title, err)
			}
		}

		verbs := []struct {
			Spanish     string
			English     string
			Example     string
			Note        string
			Pronounce   string
		}{
			{"ser/estar", "be", "\"I am happy\"", "El verbo más importante", "/biː/"},
			{"tener", "have", "\"I have a car\"", "Usado en tiempos perfectos", "/hæv/"},
			{"hacer", "do", "\"I do my homework\"", "También auxiliar en preguntas", "/duː/"},
			{"decir", "say", "\"I say hello\"", "Para expresar palabras", "/seɪ/"},
			{"ir", "go", "\"I go to school\"", "Verbo de movimiento básico", "/ɡoʊ/"},
			{"obtener", "get", "\"I get up early\"", "Muy versátil con phrasal verbs", "/ɡet/"},
			{"hacer/crear", "make", "\"Make a wish\"", "Usado para crear o elaborar", "/meɪk/"},
			{"saber/conocer", "know", "\"I know the answer\"", "Conocimiento o certeza", "/noʊ/"},
			{"pensar", "think", "\"I think so\"", "Expresa opinión o razonamiento", "/θɪŋk/"},
			{"tomar/llevar", "take", "\"Take a break\"", "Agarrar, tomar o transportar", "/teɪk/"},
		}

		for idx, vb := range verbs {
			var vocab entities.VocabularyItem
			if err := db.Where("english = ?", vb.English).First(&vocab).Error; err != nil {
				vocab = entities.VocabularyItem{
					CategoryID:      catID,
					Spanish:         vb.Spanish,
					English:         vb.English,
					Pronunciation:   stringPtr(vb.Pronounce),
					Definition:      stringPtr(vb.Note),
					ExampleSentence: stringPtr(vb.Example),
					Status:          entities.ContentStatusPublished,
				}
				if err := db.Create(&vocab).Error; err != nil {
					return fmt.Errorf("failed to seed verb %s: %w", vb.English, err)
				}
			}

			// Asociar como ítem de la lección
			var itemCount int64
			db.Model(&entities.LessonItem{}).Where("lesson_id = ? AND vocabulary_item_id = ?", lesson.ID, vocab.ID).Count(&itemCount)
			if itemCount == 0 {
				item := entities.LessonItem{
					LessonID:         lesson.ID,
					VocabularyItemID: &vocab.ID,
					ItemType:         entities.LessonContentTypeWord,
					ContentText:      stringPtr(vb.Note),
					SortOrder:        idx + 1,
					IsRequired:       true,
				}
				if err := db.Create(&item).Error; err != nil {
					return fmt.Errorf("failed to seed lesson item for verb %s: %w", vb.English, err)
				}
			}
		}
	}

	fmt.Println("Initial database seed completed successfully!")
	return nil
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
