package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestTeacherCoursePricingOnCreate(t *testing.T) {
	cases := []struct {
		name  string
		tier  entities.AccessTier
		price int
		valid bool
	}{
		{"missing tier", "", 3000, false},
		{"free", entities.AccessTierFree, 3000, false},
		{"subscription", entities.AccessTierSubscription, 3000, false},
		{"unknown tier", "other", 3000, false},
		{"zero price", entities.AccessTierCourse, 0, false},
		{"negative price", entities.AccessTierCourse, -1, false},
		{"below configured minimum", entities.AccessTierCourse, 2999, false},
		{"configured minimum", entities.AccessTierCourse, 3000, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockCourseRepo{}
			settings := &mockSettingRepo{settings: map[string]string{commands.CourseMinPriceSettingKey: "3000"}}
			course, err := commands.NewCreateCourseHandler(repo, settings).Handle(context.Background(), commands.CreateCourseCommand{
				Title: "Course", AuthorID: "teacher", OwnerType: entities.OwnerTypeTeacher, AccessTier: tc.tier, PriceCents: tc.price,
			})
			if (err == nil) != tc.valid || (repo.createdCourse != nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v, saved=%v", tc.valid, err, repo.createdCourse != nil)
			}
			if tc.valid && (course.AccessTier != tc.tier || course.PriceCents != tc.price) {
				t.Fatalf("unexpected saved pricing: %+v", course)
			}
		})
	}
}

func TestTeacherCoursePricingOnUpdate(t *testing.T) {
	free, subscription, unknown, paid := entities.AccessTierFree, entities.AccessTierSubscription, entities.AccessTier("other"), entities.AccessTierCourse
	low, valid := 2999, 3000
	cases := []struct {
		name         string
		initialTier  entities.AccessTier
		initialPrice int
		tier         *entities.AccessTier
		price        *int
		ok           bool
	}{
		{"reject free", paid, 3000, &free, nil, false},
		{"reject subscription", paid, 3000, &subscription, nil, false},
		{"reject unknown", paid, 3000, &unknown, nil, false},
		{"reject low price", paid, 3000, nil, &low, false},
		{"reject legacy invalid", free, 0, nil, nil, false},
		{"repair legacy pricing", free, 0, &paid, &valid, true},
		{"retain valid pricing", paid, 3000, nil, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			author := "teacher"
			original := &entities.Course{ID: "id", Title: "Original", CreatedBy: &author, OwnerType: entities.OwnerTypeTeacher, AccessTier: tc.initialTier, PriceCents: tc.initialPrice, Status: entities.ContentStatusDraft}
			repo := &updateMockCourseRepo{course: original}
			settings := &mockSettingRepo{settings: map[string]string{commands.CourseMinPriceSettingKey: "3000"}}
			title := "Updated"
			result, err := commands.NewUpdateCourseHandler(repo).WithSettingRepo(settings).Handle(context.Background(), commands.UpdateCourseCommand{
				CourseID: "id", Title: &title, AccessTier: tc.tier, PriceCents: tc.price, RequesterID: author, RequesterRole: entities.RoleInstructor,
			})
			if (err == nil) != tc.ok {
				t.Fatalf("ok=%v, error=%v", tc.ok, err)
			}
			if !tc.ok && (original.Title != "Original" || original.AccessTier != tc.initialTier || original.PriceCents != tc.initialPrice) {
				t.Fatal("rejected update modified course")
			}
			if tc.ok && (result.AccessTier != paid || result.PriceCents != valid) {
				t.Fatalf("unexpected pricing: %+v", result)
			}
		})
	}
}

func TestPricingPreservesPlatformAndLegacyStatusUpdates(t *testing.T) {
	author := "owner"
	status := entities.ContentStatusPublished
	legacy := &entities.Course{ID: "teacher", CreatedBy: &author, OwnerType: entities.OwnerTypeTeacher, AccessTier: entities.AccessTierFree, PriceCents: 123}
	repo := &updateMockCourseRepo{course: legacy}
	result, err := commands.NewUpdateCourseHandler(repo).Handle(context.Background(), commands.UpdateCourseCommand{
		CourseID: legacy.ID, Status: &status, RequesterID: "admin", RequesterRole: entities.RoleAdmin,
	})
	if err != nil || result.AccessTier != entities.AccessTierFree || result.PriceCents != 123 {
		t.Fatalf("status update altered legacy pricing: course=%+v error=%v", result, err)
	}

	platform := &entities.Course{ID: "platform", CreatedBy: &author, OwnerType: entities.OwnerTypePlatform, AccessTier: entities.AccessTierFree}
	repo = &updateMockCourseRepo{course: platform}
	title := "Updated platform course"
	result, err = commands.NewUpdateCourseHandler(repo).Handle(context.Background(), commands.UpdateCourseCommand{
		CourseID: platform.ID, Title: &title, RequesterID: author, RequesterRole: entities.RoleAdmin,
	})
	if err != nil || result.Title != title || result.AccessTier != entities.AccessTierFree || result.PriceCents != 0 {
		t.Fatalf("platform update changed behavior: course=%+v error=%v", result, err)
	}
}
