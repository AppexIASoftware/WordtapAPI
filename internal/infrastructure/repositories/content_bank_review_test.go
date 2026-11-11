package repositories

import (
    "context"
    "os"
    "os/exec"
    "strings"
    "testing"

    "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

const disposableCourseDSN = "root:wordtap-test-only@tcp(127.0.0.1:55943)/wordtap_course_test?charset=utf8mb4&parseTime=True&loc=Local"

func TestMySQLVaultDeleteRejectsItemUsedByPendingCourse(t *testing.T) {
    dsn := os.Getenv("WORDTAP_TEST_MYSQL_DSN")
    if dsn == "" { t.Skip("WORDTAP_TEST_MYSQL_DSN not supplied") }
    if dsn != disposableCourseDSN { t.Fatal("refusing non-disposable MySQL DSN") }
    label, err := exec.Command("docker", "inspect", "--format", `{{ index .Config.Labels "wordtap.odd.disposable" }}`, "wordtap-odd-mysql-test").Output()
    if err != nil || strings.TrimSpace(string(label)) != "true" { t.Fatalf("disposable label not verified: %q %v", label, err) }
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil { t.Fatal(err) }
    sqlDB, err := db.DB(); if err != nil { t.Fatal(err) }
    sqlDB.SetMaxOpenConns(1)
    defer sqlDB.Close()
    var schema string
    if err := db.Raw("SELECT DATABASE()").Scan(&schema).Error; err != nil || schema != "wordtap_course_test" { t.Fatalf("unsafe schema %q: %v", schema, err) }
    statements := []string{
        "CREATE TEMPORARY TABLE courses (id VARCHAR(36) PRIMARY KEY, status VARCHAR(20) NOT NULL)",
        "CREATE TEMPORARY TABLE lessons (id VARCHAR(36) PRIMARY KEY, course_id VARCHAR(36) NOT NULL)",
        "CREATE TEMPORARY TABLE lesson_items (id VARCHAR(36) PRIMARY KEY, lesson_id VARCHAR(36) NOT NULL, content_bank_item_id VARCHAR(36))",
        "CREATE TEMPORARY TABLE content_bank_items (id VARCHAR(36) PRIMARY KEY)",
        "INSERT INTO courses VALUES ('test-course', 'in_review')",
        "INSERT INTO lessons VALUES ('test-lesson', 'test-course')",
        "INSERT INTO content_bank_items VALUES ('test-item')",
        "INSERT INTO lesson_items VALUES ('test-link', 'test-lesson', 'test-item')",
    }
    for _, statement := range statements { if err := db.Exec(statement).Error; err != nil { t.Fatal(err) } }
    err = NewMySQLContentBankRepository(db).DeleteItem(context.Background(), "test-item")
    if err == nil || !strings.Contains(err.Error(), "curso en revisión") { t.Fatalf("expected review conflict, got %v", err) }
    var count int64
    if err := db.Table("content_bank_items").Where("id = ?", "test-item").Count(&count).Error; err != nil || count != 1 { t.Fatalf("protected item not preserved: count=%d err=%v", count, err) }
    if err := NewMySQLContentBankRepository(db).UpdateItem(context.Background(), &entities.ContentBankItem{ID: "test-item"}); err == nil || !strings.Contains(err.Error(), "curso en revisión") {
        t.Fatalf("expected review conflict on update, got %v", err)
    }
    if err := db.Exec("INSERT INTO content_bank_items VALUES ('unrelated-item')").Error; err != nil { t.Fatal(err) }
    if err := NewMySQLContentBankRepository(db).DeleteItem(context.Background(), "unrelated-item"); err != nil { t.Fatalf("unrelated item deletion failed: %v", err) }
    if err := db.Exec("UPDATE courses SET status = 'draft' WHERE id = 'test-course'").Error; err != nil { t.Fatal(err) }
    if err := NewMySQLContentBankRepository(db).DeleteItem(context.Background(), "test-item"); err != nil { t.Fatalf("item remains blocked after review: %v", err) }
}
