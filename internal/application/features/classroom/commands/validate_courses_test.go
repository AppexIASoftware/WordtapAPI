package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type courseLookupStub struct {
	course *entities.Course
	err    error
}

func (s courseLookupStub) FindByID(context.Context, string) (*entities.Course, error) {
	return s.course, s.err
}

func TestValidateCourses(t *testing.T) {
	owner, other := "teacher", "other"
	for _, tc := range []struct {
		name    string
		repo    CourseLookup
		allowed bool
	}{
		{"owned published", courseLookupStub{course: &entities.Course{Status: entities.ContentStatusPublished, CreatedBy: &owner}}, true},
		{"platform published", courseLookupStub{course: &entities.Course{Status: entities.ContentStatusPublished, OwnerType: entities.OwnerTypePlatform}}, true},
		{"draft", courseLookupStub{course: &entities.Course{Status: entities.ContentStatusDraft, CreatedBy: &owner}}, false},
		{"review", courseLookupStub{course: &entities.Course{Status: entities.ContentStatusInReview, CreatedBy: &owner}}, false},
		{"other teacher", courseLookupStub{course: &entities.Course{Status: entities.ContentStatusPublished, CreatedBy: &other}}, false},
		{"missing", courseLookupStub{}, false},
		{"repository error", courseLookupStub{err: errors.New("database error")}, false},
		{"missing repository", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCourses(context.Background(), tc.repo, owner, []string{"course"})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v error=%v", tc.allowed, err)
			}
		})
	}
}
