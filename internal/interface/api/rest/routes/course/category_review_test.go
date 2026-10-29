package course_test

import (
 "context"
 "errors"
 "testing"

 courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
 "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type failingCategoryReview struct { *mockCourseReviewRepository }
func (r failingCategoryReview) CreateSubmission(context.Context, *entities.CourseReviewRequest) error { return errors.New("review unavailable") }
func (r failingCategoryReview) UpdatePublishedAndSubmit(context.Context, *entities.Course, *entities.CourseReviewRequest) error { return errors.New("review unavailable") }

func TestPublishedCategoryUpdateReviewSuccess(t *testing.T) {
 owner, oldID, newID := "teacher", "old", "new"
 repo := &mockCourseRepository{courses: []entities.Course{{ID: "course", CreatedBy: &owner, OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished, CategoryID: &oldID}}}
 review := &mockCourseReviewRepository{courses: repo}
 result, err := courseCmd.NewUpdateCourseHandler(repo, review).Handle(context.Background(), courseCmd.UpdateCourseCommand{CourseID: "course", RequesterID: owner, RequesterRole: entities.RoleInstructor, CategoryID: &newID, CategoryIDSet: true})
 if err != nil { t.Fatal(err) }
 if result.Status != entities.ContentStatusInReview || repo.courses[0].Status != entities.ContentStatusInReview || *repo.courses[0].CategoryID != newID || len(review.requests) != 1 { t.Fatalf("expected saved change and review: course=%+v requests=%d", repo.courses[0], len(review.requests)) }
}

func TestPublishedCategoryUpdateWithoutReviewRepository(t *testing.T) {
 owner, oldID, newID := "teacher", "old", "new"
 repo := &mockCourseRepository{courses: []entities.Course{{ID: "course", CreatedBy: &owner, OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished, CategoryID: &oldID}}}
 _, err := courseCmd.NewUpdateCourseHandler(repo).Handle(context.Background(), courseCmd.UpdateCourseCommand{CourseID: "course", RequesterID: owner, RequesterRole: entities.RoleInstructor, CategoryID: &newID, CategoryIDSet: true})
 if err == nil || repo.courses[0].Status != entities.ContentStatusPublished || *repo.courses[0].CategoryID != oldID { t.Fatalf("expected no write when review unavailable: err=%v course=%+v", err, repo.courses[0]) }
}

func TestPublishedCategoryUpdateReviewFailure(t *testing.T) {
 owner, oldID, newID := "teacher", "old", "new"
 repo := &mockCourseRepository{courses: []entities.Course{{ID: "course", CreatedBy: &owner, OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished, CategoryID: &oldID}}}
 review := failingCategoryReview{&mockCourseReviewRepository{courses: repo}}
 result, err := courseCmd.NewUpdateCourseHandler(repo, review).Handle(context.Background(), courseCmd.UpdateCourseCommand{CourseID: "course", RequesterID: owner, RequesterRole: entities.RoleInstructor, CategoryID: &newID, CategoryIDSet: true})
 if err == nil { t.Fatalf("expected error, got success %+v", result) }
 if repo.courses[0].Status != entities.ContentStatusPublished || *repo.courses[0].CategoryID != oldID { t.Fatalf("failed review must not persist update: %+v", repo.courses[0]) }
}
