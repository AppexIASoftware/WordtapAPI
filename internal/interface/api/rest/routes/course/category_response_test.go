package course_test

import (
 "context"
 "errors"
 "testing"

 courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
 "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type categoryLoadingRepo struct { *mockCourseRepository }

func (r categoryLoadingRepo) FindByID(ctx context.Context, id string) (*entities.Course, error) {
 course, err := r.mockCourseRepository.FindByID(ctx, id)
 if err == nil && course.CategoryID != nil {
  course.Category = &entities.ContentCategory{ID: *course.CategoryID, Name: "Updated category"}
 }
 return course, err
}

type failingReloadRepo struct {
 *mockCourseRepository
 calls int
}

func (r *failingReloadRepo) FindByID(ctx context.Context, id string) (*entities.Course, error) {
 r.calls++
 if r.calls > 1 { return nil, errors.New("reload failed") }
 return r.mockCourseRepository.FindByID(ctx, id)
}

func TestCourseUpdateSucceedsWhenCategoryReloadFails(t *testing.T) {
 owner, newCategory := "teacher", "new"
 repo := &failingReloadRepo{mockCourseRepository: &mockCourseRepository{courses: []entities.Course{{ID: "course", CreatedBy: &owner, OwnerType: entities.OwnerTypeTeacher}}}}
 result, err := courseCmd.NewUpdateCourseHandler(repo).Handle(context.Background(), courseCmd.UpdateCourseCommand{CourseID: "course", RequesterID: owner, RequesterRole: entities.RoleInstructor, CategoryID: &newCategory, CategoryIDSet: true})
 if err != nil { t.Fatalf("persisted update must not report failure: %v", err) }
 if result.CategoryID == nil || *result.CategoryID != newCategory || repo.courses[0].CategoryID == nil || *repo.courses[0].CategoryID != newCategory { t.Fatal("updated category not returned and persisted") }
}

func TestUpdatedCourseIncludesCategory(t *testing.T) {
 owner, oldCategory, newCategory := "teacher", "old", "new"
 repo := categoryLoadingRepo{&mockCourseRepository{courses: []entities.Course{{ID:"course", CreatedBy:&owner, OwnerType:entities.OwnerTypeTeacher, CategoryID:&oldCategory}}}}
 result, err := courseCmd.NewUpdateCourseHandler(repo).Handle(context.Background(), courseCmd.UpdateCourseCommand{CourseID:"course", RequesterID:owner, RequesterRole:entities.RoleInstructor, CategoryID:&newCategory, CategoryIDSet:true})
 if err != nil { t.Fatal(err) }
 if result.Category == nil || result.Category.ID != newCategory { t.Fatalf("updated category missing from response: %+v", result.Category) }
}
