package access

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type testPurchases struct{ err error }

func (p testPurchases) HasPurchase(_ context.Context, userID, courseID string) (bool, error) {
	return userID == "buyer" && courseID == "teacher-course", p.err
}

func TestContentPolicy(t *testing.T) {
	owner := "teacher"
	course := entities.Course{ID: "teacher-course", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished, CreatedBy: &owner}
	lesson := entities.Lesson{CourseID: course.ID, Status: entities.ContentStatusPublished}
	for _, tc := range []struct {
		name    string
		user    string
		role    entities.UserRole
		course  entities.Course
		lesson  entities.Lesson
		enabled bool
		want    error
	}{
		{"purchaser independent of classroom", "buyer", entities.RoleStudent, course, lesson, true, nil},
		{"other student", "other", entities.RoleStudent, course, lesson, true, ErrContentDenied},
		{"foreign instructor", "other-teacher", entities.RoleInstructor, course, lesson, true, ErrContentDenied},
		{"simulation disabled", "buyer", entities.RoleStudent, course, lesson, false, ErrContentDenied},
		{"owner", "teacher", entities.RoleInstructor, course, lesson, false, nil},
		{"administrator", "admin", entities.RoleAdmin, course, lesson, false, nil},
		{"moderator", "moderator", entities.RoleModerator, course, lesson, false, nil},
		{"anonymous", "", entities.RoleStudent, course, lesson, true, ErrContentDenied},
		{"draft lesson", "buyer", entities.RoleStudent, course, entities.Lesson{CourseID: course.ID, Status: entities.ContentStatusDraft}, true, ErrContentDenied},
		{"other course lesson", "buyer", entities.RoleStudent, course, entities.Lesson{CourseID: "other", Status: entities.ContentStatusPublished}, true, ErrContentDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := NewContentPolicy(testPurchases{}, tc.enabled).Authorize(context.Background(), &tc.course, &tc.lesson, tc.user, tc.role)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	platform := course
	platform.OwnerType = entities.OwnerTypePlatform
	if err := NewContentPolicy(nil, false).Authorize(context.Background(), &platform, &lesson, "other", entities.RoleStudent); err != nil {
		t.Fatal(err)
	}
	dbError := errors.New("database unavailable")
	if err := NewContentPolicy(testPurchases{err: dbError}, true).Authorize(context.Background(), &course, &lesson, "buyer", entities.RoleStudent); !errors.Is(err, dbError) {
		t.Fatalf("got %v", err)
	}

	ent := testEntitlements{subUser: "pro-student", courseUser: "real-buyer", courseID: course.ID}
	subPlatform := platform
	subPlatform.AccessTier = entities.AccessTierSubscription
	if err := NewContentPolicy(nil, false).WithEntitlements(ent).Authorize(context.Background(), &subPlatform, &lesson, "free-student", entities.RoleStudent); !errors.Is(err, ErrContentDenied) {
		t.Fatalf("expected free student denied subscription platform course, got %v", err)
	}
	if err := NewContentPolicy(nil, false).WithEntitlements(ent).Authorize(context.Background(), &subPlatform, &lesson, "pro-student", entities.RoleStudent); err != nil {
		t.Fatalf("expected active subscriber allowed on platform subscription course, got %v", err)
	}
	paidCourse := course
	paidCourse.AccessTier = entities.AccessTierCourse
	if err := NewContentPolicy(nil, false).WithEntitlements(ent).Authorize(context.Background(), &paidCourse, &lesson, "real-buyer", entities.RoleStudent); err != nil {
		t.Fatalf("expected real course entitlement to allow access even with simulation disabled, got %v", err)
	}
	paidPlatform := platform
	paidPlatform.AccessTier = entities.AccessTierCourse
	if err := NewContentPolicy(nil, false).WithEntitlements(ent).Authorize(context.Background(), &paidPlatform, &lesson, "free-student", entities.RoleStudent); !errors.Is(err, ErrContentDenied) {
		t.Fatalf("expected unentitled student denied paid platform course, got %v", err)
	}
	freePriceMismatch := course
	freePriceMismatch.AccessTier = entities.AccessTierFree
	freePriceMismatch.PriceCents = 1500
	if err := NewContentPolicy(nil, false).Authorize(context.Background(), &freePriceMismatch, &lesson, "free-student", entities.RoleStudent); !errors.Is(err, ErrContentDenied) {
		t.Fatalf("expected teacher free course with PriceCents > 0 denied, got %v", err)
	}
}

type testEntitlements struct {
	subUser    string
	courseUser string
	courseID   string
	err        error
}

func (e testEntitlements) HasActiveSubscription(_ context.Context, userID string) (bool, error) {
	return userID == e.subUser, e.err
}

func (e testEntitlements) HasCourseEntitlement(_ context.Context, userID, courseID string) (bool, error) {
	return userID == e.courseUser && courseID == e.courseID, e.err
}
