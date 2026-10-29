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
		&entities.TeacherApplication{},
		&entities.PlatformSetting{},

		// Catálogo y Aprendizaje
		&entities.ContentCategory{},
		&entities.ContentBankType{},
		&entities.ContentBankItem{},
		&entities.Course{},
		&entities.CourseReviewRequest{},
		&entities.Lesson{},
		&entities.VocabularyItem{},
		&entities.Phrase{},
		&entities.LessonItem{},
		&entities.ContentAsset{},

		// Aulas y Cohortes Docente
		&entities.Classroom{},
		&entities.ClassroomEnrollment{},
		&entities.StudentStumble{},

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
		&entities.TeacherGame{},
		&entities.LessonGame{},

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

	// 1.b. Relleno de propiedad para datos existentes
	// Las actividades globales pertenecen a la plataforma; el resto conserva
	// la propiedad del docente. Los cursos creados por administradores son
	// contenido de plataforma. Las sentencias son idempotentes.
	if err := db.Exec(
		`UPDATE teacher_games SET owner_type = ? WHERE is_global = ? AND owner_type <> ?`,
		entities.OwnerTypePlatform, true, entities.OwnerTypePlatform,
	).Error; err != nil {
		return fmt.Errorf("failed to backfill platform game ownership: %w", err)
	}
	if err := db.Exec(
		`UPDATE courses c JOIN users u ON u.id = c.created_by
		 SET c.owner_type = ?
		 WHERE u.role = ? AND c.owner_type <> ?`,
		entities.OwnerTypePlatform, entities.RoleAdmin, entities.OwnerTypePlatform,
	).Error; err != nil {
		return fmt.Errorf("failed to backfill platform course ownership: %w", err)
	}
	if err := db.Exec(
		`UPDATE teacher_games SET owner_type = ? WHERE owner_type IS NULL OR owner_type = ''`,
		entities.OwnerTypeTeacher,
	).Error; err != nil {
		return fmt.Errorf("failed to backfill teacher game ownership: %w", err)
	}
	if err := db.Exec(
		`UPDATE teacher_games SET access_tier = ? WHERE access_tier IS NULL OR access_tier = ''`,
		entities.AccessTierFree,
	).Error; err != nil {
		return fmt.Errorf("failed to backfill teacher game access tier: %w", err)
	}
	if err := db.Exec(
		`UPDATE courses SET owner_type = ? WHERE owner_type IS NULL OR owner_type = ''`,
		entities.OwnerTypeTeacher,
	).Error; err != nil {
		return fmt.Errorf("failed to backfill course ownership: %w", err)
	}
	if err := db.Exec(
		`UPDATE lessons l JOIN courses c ON l.course_id = c.id
		 SET l.status = ?
		 WHERE c.status = ? AND l.status <> ?`,
		entities.ContentStatusPublished, entities.ContentStatusPublished, entities.ContentStatusPublished,
	).Error; err != nil {
		return fmt.Errorf("failed to sync published lesson status: %w", err)
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

	// 3. Índices compuestos de alto rendimiento para eliminar filesort y escaneo de tablas
	indexes := []string{
		`CREATE INDEX idx_courses_author_created ON courses (created_by, created_at DESC)`,
		`CREATE INDEX idx_teacher_apps_status_created ON teacher_applications (status, created_at DESC)`,
		`CREATE INDEX idx_course_reviews_instructor ON course_review_requests (instructor_id, submitted_at DESC)`,
		`CREATE INDEX idx_lessons_course_order_created ON lessons (course_id, sort_order ASC, created_at ASC)`,
		`CREATE INDEX idx_items_lesson_order ON lesson_items (lesson_id, sort_order ASC)`,
	}
	for _, idx := range indexes {
		_ = db.Exec(idx)
	}

	fmt.Println("Database migration completed successfully!")
	return nil
}
