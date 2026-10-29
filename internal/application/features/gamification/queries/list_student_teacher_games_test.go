package queries_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockStudentClassroomRepo struct {
	classrooms []entities.Classroom
}

func (m *mockStudentClassroomRepo) Create(ctx context.Context, c *entities.Classroom) error {
	return nil
}
func (m *mockStudentClassroomRepo) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepo) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepo) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	var res []entities.Classroom
	for _, c := range m.classrooms {
		if c.IsActive {
			for _, e := range c.Enrollments {
				if e.StudentID == studentID {
					res = append(res, c)
					break
				}
			}
		}
	}
	return res, nil
}
func (m *mockStudentClassroomRepo) EnrollStudent(ctx context.Context, e *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockStudentClassroomRepo) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepo) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepo) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepo) CreateStumble(ctx context.Context, s *entities.StudentStumble) error {
	return nil
}
func (m *mockStudentClassroomRepo) Update(ctx context.Context, c *entities.Classroom) error {
	return nil
}

func TestListStudentTeacherGamesHandler(t *testing.T) {
	ctx := context.Background()

	courseA := "course-A"
	courseB := "course-B"
	assignedCourseJSON := `["course-A"]`
	assignedCourseOtherJSON := `["course-X"]`

	classrooms := []entities.Classroom{
		{
			ID:        "cls-1",
			TeacherID: "teacher-1",
			CourseID:  &courseA,
			IsActive:  true,
			Enrollments: []entities.ClassroomEnrollment{
				{StudentID: "student-1"},
			},
		},
		{
			ID:              "cls-2",
			TeacherID:       "teacher-2",
			AssignedCourses: &courseB,
			IsActive:        false, // Inactiva (revocada)
			Enrollments: []entities.ClassroomEnrollment{
				{StudentID: "student-1"},
			},
		},
	}

	games := []entities.TeacherGame{
		// Teacher 1: publicado standalone para todos los alumnos
		{
			ID:                    "game-1",
			TeacherID:             "teacher-1",
			Title:                 "Juego General Teacher 1",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "all_students",
		},
		// Teacher 1: publicado standalone solo para alumnos de course-A (alumno inscrito en course-A -> acceso concedido)
		{
			ID:                    "game-2",
			TeacherID:             "teacher-1",
			Title:                 "Juego Específico Course A",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "selected_courses",
			AssignedCourseIDs:     &assignedCourseJSON,
		},
		// Teacher 1: publicado standalone solo para alumnos de course-X (alumno NO inscrito en course-X -> acceso denegado)
		{
			ID:                    "game-3",
			TeacherID:             "teacher-1",
			Title:                 "Juego Específico Course X",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "selected_courses",
			AssignedCourseIDs:     &assignedCourseOtherJSON,
		},
		// Teacher 1: NO publicado standalone (solo lecciones -> alumno NO debe verlo en catálogo app)
		{
			ID:                    "game-4",
			TeacherID:             "teacher-1",
			Title:                 "Juego Oculto Solo Lecciones",
			IsActive:              true,
			IsPublishedStandalone: false,
			AudienceScope:         "all_students",
		},
		// Teacher 2: publicado standalone, pero aula inactiva -> acceso revocado
		{
			ID:                    "game-5",
			TeacherID:             "teacher-2",
			Title:                 "Juego Teacher 2",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "all_students",
		},
		// Teacher 3: publicado standalone, pero alumno NO está inscrito con Teacher 3
		{
			ID:                    "game-6",
			TeacherID:             "teacher-3",
			Title:                 "Juego Teacher 3 Ajeno",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "all_students",
		},
	}

	gameRepo := &mockQueryGameRepo{games: games}
	classroomRepo := &mockStudentClassroomRepo{classrooms: classrooms}

	handler := queries.NewListStudentTeacherGamesHandler(classroomRepo, gameRepo)

	t.Run("alumno lista exclusivamente juegos autorizados de sus docentes", func(t *testing.T) {
		res, err := handler.Handle(ctx, queries.ListStudentTeacherGamesQuery{
			StudentID: "student-1",
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}

		if len(res) != 2 {
			t.Fatalf("se esperaban 2 juegos autorizados (game-1 y game-2), se obtuvieron %d", len(res))
		}

		foundGame1 := false
		foundGame2 := false
		for _, g := range res {
			if g.ID == "game-1" {
				foundGame1 = true
			}
			if g.ID == "game-2" {
				foundGame2 = true
			}
			if g.ID == "game-3" || g.ID == "game-4" || g.ID == "game-5" || g.ID == "game-6" {
				t.Errorf("juego no autorizado filtrado erróneamente: %s", g.ID)
			}
		}

		if !foundGame1 || !foundGame2 {
			t.Errorf("no se encontraron los juegos esperados game-1 y game-2")
		}
	})

	t.Run("alumno sin inscripciones activas obtiene lista vacía", func(t *testing.T) {
		res, err := handler.Handle(ctx, queries.ListStudentTeacherGamesQuery{
			StudentID: "student-1",
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("se esperaban 2 juegos autorizados, se obtuvieron %d", len(res))
		}
	})
}

func TestGetStudentTeacherGameDetailHandler(t *testing.T) {
	ctx := context.Background()

	courseA := "course-A"
	assignedCourseJSON := `["course-A"]`
	assignedCourseOtherJSON := `["course-X"]`

	classrooms := []entities.Classroom{
		{
			ID:        "cls-1",
			TeacherID: "teacher-1",
			CourseID:  &courseA,
			IsActive:  true,
			Enrollments: []entities.ClassroomEnrollment{
				{StudentID: "student-1"},
			},
		},
	}

	games := []entities.TeacherGame{
		{
			ID:                    "game-1",
			TeacherID:             "teacher-1",
			Title:                 "Juego General Teacher 1",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "all_students",
		},
		{
			ID:                    "game-2",
			TeacherID:             "teacher-1",
			Title:                 "Juego Específico Course A",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "selected_courses",
			AssignedCourseIDs:     &assignedCourseJSON,
		},
		{
			ID:                    "game-3",
			TeacherID:             "teacher-1",
			Title:                 "Juego Específico Course X",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "selected_courses",
			AssignedCourseIDs:     &assignedCourseOtherJSON,
		},
		{
			ID:                    "game-4",
			TeacherID:             "teacher-1",
			Title:                 "Juego Oculto Solo Lecciones",
			IsActive:              true,
			IsPublishedStandalone: false,
			AudienceScope:         "all_students",
		},
	}

	gameRepo := &mockQueryGameRepo{games: games}
	classroomRepo := &mockStudentClassroomRepo{classrooms: classrooms}

	handler := queries.NewGetStudentTeacherGameDetailHandler(classroomRepo, gameRepo)

	t.Run("acceso permitido a juego general", func(t *testing.T) {
		g, err := handler.Handle(ctx, queries.GetStudentTeacherGameDetailQuery{
			StudentID: "student-1",
			GameID:    "game-1",
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if g.ID != "game-1" {
			t.Errorf("esperado game-1, obtenido %s", g.ID)
		}
	})

	t.Run("acceso permitido a juego de curso inscrito", func(t *testing.T) {
		g, err := handler.Handle(ctx, queries.GetStudentTeacherGameDetailQuery{
			StudentID: "student-1",
			GameID:    "game-2",
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if g.ID != "game-2" {
			t.Errorf("esperado game-2, obtenido %s", g.ID)
		}
	})

	t.Run("acceso denegado a juego de curso no inscrito", func(t *testing.T) {
		_, err := handler.Handle(ctx, queries.GetStudentTeacherGameDetailQuery{
			StudentID: "student-1",
			GameID:    "game-3",
		})
		if err == nil {
			t.Fatal("se esperaba error de acceso denegado")
		}
	})

	t.Run("acceso denegado a juego no publicado standalone", func(t *testing.T) {
		_, err := handler.Handle(ctx, queries.GetStudentTeacherGameDetailQuery{
			StudentID: "student-1",
			GameID:    "game-4",
		})
		if err == nil {
			t.Fatal("se esperaba error de no disponible")
		}
	})

	t.Run("acceso denegado a alumno no inscrito", func(t *testing.T) {
		_, err := handler.Handle(ctx, queries.GetStudentTeacherGameDetailQuery{
			StudentID: "student-ajeno",
			GameID:    "game-1",
		})
		if err == nil {
			t.Fatal("se esperaba error de no permisos")
		}
	})
}
