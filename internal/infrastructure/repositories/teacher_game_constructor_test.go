package repositories

import (
	"testing"

	"gorm.io/gorm"
)

func TestNewMySQLTeacherGameRepositoryDoesNotAccessDatabase(t *testing.T) {
	db := &gorm.DB{}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("constructor accessed database: %v", recovered)
		}
	}()
	if got := NewMySQLTeacherGameRepository(db); got == nil {
		t.Fatal("expected a repository")
	}
}

func TestNewMySQLTeacherGameRepositoryAcceptsNil(t *testing.T) {
	if got := NewMySQLTeacherGameRepository(nil); got == nil {
		t.Fatal("expected a repository")
	}
}
