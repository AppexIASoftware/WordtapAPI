package repositories

import (
	"sync"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"gorm.io/gorm/schema"
)

func TestHistoryReviewerAssociationMatchesSchema(t *testing.T) {
	parsed, err := schema.Parse(&entities.CourseReviewRequest{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Relationships.Relations[courseReviewReviewerAssociation] == nil {
		t.Fatalf("history preload %q is not a CourseReviewRequest association", courseReviewReviewerAssociation)
	}
}
