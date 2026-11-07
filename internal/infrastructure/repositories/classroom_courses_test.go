package repositories

import (
	"reflect"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestAssignedCourseIDs(t *testing.T) {
	primary := "course-1"
	assigned := "course-1, course-2, ,course-2,course-3"
	got := assignedCourseIDs(entities.Classroom{CourseID: &primary, AssignedCourses: &assigned})
	want := []string{"course-1", "course-2", "course-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assignedCourseIDs() = %v; want %v", got, want)
	}
}
