package repositories

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// This test never creates a schema or runs a migration. The opt-in DSN must point
// to a dedicated *_test database with the application schema already installed.
func TestSimulatedCoursePurchaseMySQL(t *testing.T) {
	dsn := os.Getenv("SIMULATED_PURCHASE_TEST_DSN")
	if dsn == "" {
		t.Skip("SIMULATED_PURCHASE_TEST_DSN is not configured")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var database string
	if err := db.Raw("SELECT DATABASE()").Scan(&database).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(database, "_test") {
		t.Fatal("refusing to write to a database without the _test suffix")
	}
	studentID, courseID, otherID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	student := entities.User{ID: studentID, Email: studentID + "@example.invalid", Name: "Test student", Role: entities.RoleStudent, IsActive: true}
	if err := db.Create(&student).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM simulated_course_purchase_keys WHERE user_id = ?", studentID)
		db.Exec("DELETE FROM simulated_course_purchases WHERE user_id = ?", studentID)
		db.Exec("DELETE FROM courses WHERE id IN (?, ?)", courseID, otherID)
		db.Exec("DELETE FROM users WHERE id = ?", studentID)
	})
	for _, id := range []string{courseID, otherID} {
		course := entities.Course{ID: id, Title: "Purchase test", Slug: id, OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished, AccessTier: entities.AccessTierCourse, PriceCents: 2750}
		if err := db.Create(&course).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewMySQLSimulatedPurchaseRepository(db)
	const workers = 8
	results := make(chan *commands.SimulatedCoursePurchase, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := repo.Purchase(context.Background(), studentID, courseID, fmt.Sprintf("key-%d", i))
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}(i)
	}
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		t.Error(err)
	}
	var id string
	count := 0
	for p := range results {
		count++
		if id == "" {
			id = p.ID
		}
		if p.ID != id || p.AmountCents != 2750 || p.Currency != "PEN" {
			t.Errorf("unexpected result: %+v", p)
		}
	}
	if count != workers {
		t.Fatalf("%d of %d purchases completed", count, workers)
	}
	if _, err := repo.Purchase(context.Background(), studentID, otherID, "key-0"); err != commands.ErrIdempotencyConflict {
		t.Fatalf("expected key conflict, got %v", err)
	}
	if err := db.Model(&entities.Course{}).Where("id = ?", courseID).Update("price_cents", 9999).Error; err != nil {
		t.Fatal(err)
	}
	got, err := repo.Purchase(context.Background(), studentID, courseID, "key-0")
	if err != nil || got == nil || got.AmountCents != 2750 {
		t.Fatalf("retry did not retain original price: %+v %v", got, err)
	}
	got, err = repo.Find(context.Background(), studentID, courseID)
	if err != nil || got == nil || got.ID != id {
		t.Fatalf("entitlement lookup: %+v %v", got, err)
	}
}
