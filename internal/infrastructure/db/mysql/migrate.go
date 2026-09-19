package mysql

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// AutoMigrate ejecuta la migración de tablas y vistas en MySQL.
func AutoMigrate(db *gorm.DB) error {
	fmt.Println("Running database migrations...")

	err := db.AutoMigrate(
		// Autenticación y Usuario
		&entities.User{},
		&entities.UserAuthAccount{},
		&entities.UserSession{},
		&entities.AccountToken{},
		&entities.UserDevice{},

		// Catálogo y Aprendizaje
		&entities.ContentCategory{},
		&entities.Course{},
		&entities.Lesson{},
		&entities.VocabularyItem{},
		&entities.Phrase{},
		&entities.LessonItem{},
		&entities.ContentAsset{},

		// Progreso y Estadísticas
		&entities.UserLessonProgress{},
		&entities.UserVocabulary{},
		&entities.UserLearningStats{},
		&entities.UserDailyActivity{},

		// Juegos
		&entities.Game{},
		&entities.GameQuestion{},
		&entities.GameAttempt{},
		&entities.GameAnswer{},
		&entities.UserGameUnlock{},

		// Diccionario y Notificaciones
		&entities.TranslationEntry{},
		&entities.UserSearchHistory{},
		&entities.Notification{},
		&entities.NotificationPreference{},
		&entities.NotificationSchedule{},

		// Productos y Facturación
		&entities.Product{},
		&entities.Order{},
		&entities.UserEntitlement{},
		&entities.Subscription{},
		&entities.PaymentEvent{},

		// Gamificación y Administración
		&entities.Achievement{},
		&entities.UserAchievement{},
		&entities.DailyChallenge{},
		&entities.UserDailyChallenge{},
		&entities.LeaderboardScore{},
		&entities.AdminRole{},
		&entities.UserAdminRole{},
		&entities.AuditLog{},
		&entities.ContentReport{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate database tables: %w", err)
	}

	// 2. Vistas SQL de lectura optimizada
	views := []string{
		// v_user_dashboard: Unifica perfil y estadísticas del usuario para alimentar la pantalla Home en una sola lectura.
		`CREATE OR REPLACE VIEW v_user_dashboard AS
		SELECT
			u.id AS user_id,
			u.name,
			u.email,
			u.access_tier,
			COALESCE(s.words_learned, 0) AS words_learned,
			COALESCE(s.lessons_completed, 0) AS lessons_completed,
			COALESCE(s.games_completed, 0) AS games_completed,
			COALESCE(s.total_stars, 0) AS total_stars,
			COALESCE(s.total_points, 0) AS total_points,
			COALESCE(s.current_streak_days, 0) AS current_streak_days,
			COALESCE(s.accuracy_percent, 0) AS accuracy_percent
		FROM users u
		LEFT JOIN user_learning_stats s ON s.user_id = u.id
		WHERE u.is_active = TRUE;`,

		// v_published_learning_content: Lista solo lecciones publicadas con el nombre de su categoría para el catálogo.
		`CREATE OR REPLACE VIEW v_published_learning_content AS
		SELECT
			l.id AS lesson_id,
			l.course_id,
			l.title AS lesson_title,
			l.access_tier,
			l.sort_order,
			c.name AS category_name
		FROM lessons l
		LEFT JOIN content_categories c ON c.id = l.category_id
		WHERE l.status = 'published';`,

		// v_active_premium_access: Verifica en tiempo real si el usuario tiene acceso premium o suscripción vigente.
		`CREATE OR REPLACE VIEW v_active_premium_access AS
		SELECT
			e.user_id,
			e.product_id,
			e.entitlement_type,
			e.starts_at,
			e.ends_at
		FROM user_entitlements e
		WHERE e.is_active = TRUE
			AND e.starts_at <= NOW()
			AND (e.ends_at IS NULL OR e.ends_at > NOW());`,
	}

	for _, v := range views {
		if err := db.Exec(v).Error; err != nil {
			return fmt.Errorf("failed to create view: %w", err)
		}
	}

	fmt.Println("Database migration completed successfully!")
	return nil
}
