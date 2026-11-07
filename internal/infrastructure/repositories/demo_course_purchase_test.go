package repositories

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
)

func TestDemoPurchaseGrantsAccessForOneStudentAndCourse(t *testing.T) {
	repo := NewDemoPurchaseRepository()
	ctx := context.Background()

	if has, err := repo.HasPurchase(ctx, "student-1", "course-1"); err != nil || has {
		t.Fatalf("initial access = %v, %v; want false, nil", has, err)
	}

	created, err := repo.Purchase(ctx, "student-1", "course-1", "key-1")
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if created.ID == "" || created.CourseID != "course-1" || created.UserID != "student-1" {
		t.Fatalf("unexpected receipt: %+v", created)
	}
	if created.AmountCents != 0 {
		t.Fatalf("demo access must not represent a payment, amount = %d", created.AmountCents)
	}

	found, err := repo.Find(ctx, "student-1", "course-1")
	if err != nil || found == nil || found.ID != created.ID {
		t.Fatalf("find = %+v, %v; want receipt %s", found, err, created.ID)
	}
	if has, err := repo.HasPurchase(ctx, "student-1", "course-1"); err != nil || !has {
		t.Fatalf("access after grant = %v, %v; want true, nil", has, err)
	}

	if has, _ := repo.HasPurchase(ctx, "student-2", "course-1"); has {
		t.Fatal("access leaked to another student")
	}
	if has, _ := repo.HasPurchase(ctx, "student-1", "course-2"); has {
		t.Fatal("access leaked to another course")
	}
	if found, _ := repo.Find(ctx, "student-1", "course-2"); found != nil {
		t.Fatal("find returned a receipt for an ungranted course")
	}
}

func TestDemoPurchaseRejectsKeyReusedForAnotherCourse(t *testing.T) {
	repo := NewDemoPurchaseRepository()
	ctx := context.Background()

	if _, err := repo.Purchase(ctx, "student-1", "course-1", "shared-key"); err != nil {
		t.Fatalf("first purchase: %v", err)
	}
	if _, err := repo.Purchase(ctx, "student-1", "course-2", "shared-key"); !errors.Is(err, commands.ErrIdempotencyConflict) {
		t.Fatalf("error = %v; want ErrIdempotencyConflict", err)
	}
	if has, _ := repo.HasPurchase(ctx, "student-1", "course-2"); has {
		t.Fatal("conflicting request granted access")
	}
}

func TestDemoPurchaseIsRaceFree(t *testing.T) {
	repo := NewDemoPurchaseRepository()
	ctx := context.Background()

	const workers = 24
	ids := make([]string, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			receipt, err := repo.Purchase(ctx, "student-1", "course-1", fmt.Sprintf("key-%d", index))
			if err != nil {
				t.Errorf("purchase %d: %v", index, err)
				return
			}
			ids[index] = receipt.ID
		}(i)
	}
	wg.Wait()

	for i := 1; i < workers; i++ {
		if ids[i] != ids[0] {
			t.Fatalf("concurrent purchases created different receipts: %s != %s", ids[i], ids[0])
		}
	}
}
