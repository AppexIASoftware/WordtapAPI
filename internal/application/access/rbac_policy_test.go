package access

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type stubSettingRepo struct {
	values map[string]string
	err    error
}

func (s *stubSettingRepo) Get(_ context.Context, key string) (*entities.PlatformSetting, error) {
	if s.err != nil {
		return nil, s.err
	}
	v, ok := s.values[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &entities.PlatformSetting{Key: key, Value: v}, nil
}

func (s *stubSettingRepo) Set(_ context.Context, key string, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *stubSettingRepo) GetAll(_ context.Context) ([]entities.PlatformSetting, error) {
	return nil, nil
}

func TestRBACPolicyEvaluator_DefaultsAndOverrides(t *testing.T) {
	ctx := context.Background()

	t.Run("uses defaults when repository is nil or empty", func(t *testing.T) {
		eval := NewRBACPolicyEvaluator(nil)
		if !eval.IsAllowed(ctx, entities.RoleAdmin, PermCoursesAuthorFreeCourses) {
			t.Fatal("admin should always be allowed")
		}
		if eval.IsAllowed(ctx, entities.RoleStudent, PermCoursesCreateDrafts) {
			t.Fatal("student should never be allowed")
		}
		if eval.IsAllowed(ctx, entities.RoleModerator, PermCoursesCreateDrafts) {
			t.Fatal("moderator courses:create_drafts should default to false")
		}
		if !eval.IsAllowed(ctx, entities.RoleModerator, PermModerationReviewCourses) {
			t.Fatal("moderator moderation:review_courses should default to true")
		}
	})

	t.Run("applies configurable overrides from platform_settings", func(t *testing.T) {
		repo := &stubSettingRepo{
			values: map[string]string{
				RBACSettingKey: `{
					"courses:create_drafts": {"moderator": true},
					"courses:author_free_courses": {"instructor": true},
					"moderation:review_courses": {"moderator": false},
					"monetization:custom_course_price": {"instructor": false}
				}`,
			},
		}
		eval := NewRBACPolicyEvaluator(repo)

		if !eval.IsAllowed(ctx, entities.RoleModerator, PermCoursesCreateDrafts) {
			t.Fatal("expected moderator courses:create_drafts override to be true")
		}
		if !eval.IsAllowed(ctx, entities.RoleInstructor, PermCoursesAuthorFreeCourses) {
			t.Fatal("expected instructor courses:author_free_courses override to be true")
		}
		if eval.IsAllowed(ctx, entities.RoleModerator, PermModerationReviewCourses) {
			t.Fatal("expected moderator moderation:review_courses override to be false")
		}
		if eval.IsAllowed(ctx, entities.RoleInstructor, PermMonetizationCustomCoursePrice) {
			t.Fatal("expected instructor monetization:custom_course_price override to be false")
		}
	})

	t.Run("ignores overrides on locked structural invariants", func(t *testing.T) {
		repo := &stubSettingRepo{
			values: map[string]string{
				RBACSettingKey: `{
					"courses:create_drafts": {"instructor": false},
					"moderation:review_courses": {"instructor": true},
					"monetization:custom_course_price": {"moderator": true}
				}`,
			},
		}
		eval := NewRBACPolicyEvaluator(repo)

		if !eval.IsAllowed(ctx, entities.RoleInstructor, PermCoursesCreateDrafts) {
			t.Fatal("locked invariant instructor courses:create_drafts must remain true")
		}
		if eval.IsAllowed(ctx, entities.RoleInstructor, PermModerationReviewCourses) {
			t.Fatal("locked invariant instructor moderation:review_courses must remain false")
		}
		if eval.IsAllowed(ctx, entities.RoleModerator, PermMonetizationCustomCoursePrice) {
			t.Fatal("locked invariant moderator monetization:custom_course_price must remain false")
		}
	})
}
