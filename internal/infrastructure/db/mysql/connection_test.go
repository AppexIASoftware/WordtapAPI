package mysql

import (
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestNormalizeDSN(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "adds default query params when absent",
			input:    "user:pass@tcp(localhost:3306)/wordtap",
			contains: []string{"charset=utf8mb4", "parseTime=True", "loc=UTC", "interpolateParams=true"},
		},
		{
			name:     "overrides loc=Local and injects missing parseTime/interpolateParams",
			input:    "user:pass@tcp(db:3306)/wordtap?charset=utf8mb4&loc=Local",
			contains: []string{"charset=utf8mb4", "parseTime=True", "loc=UTC", "interpolateParams=true"},
			excludes: []string{"loc=Local"},
		},
		{
			name:     "preserves explicit interpolateParams=false and timeout while enforcing UTC",
			input:    "user:pass@tcp(db:3306)/wordtap?interpolateParams=false&timeout=5s&loc=America%2FBogota",
			contains: []string{"charset=utf8mb4", "interpolateParams=false", "timeout=5s", "loc=UTC", "parseTime=True"},
			excludes: []string{"Bogota"},
		},
		{
			name:     "keeps question mark inside password untouched when splitting query",
			input:    "user:p?ss@tcp(db:3306)/wordtap?loc=Local",
			contains: []string{"user:p?ss@tcp(db:3306)/wordtap?", "charset=utf8mb4", "loc=UTC", "parseTime=True", "interpolateParams=true"},
			excludes: []string{"loc=Local"},
		},
		{
			name:     "keeps question mark inside password untouched without query string",
			input:    "user:p?ss@tcp(db:3306)/wordtap",
			contains: []string{"user:p?ss@tcp(db:3306)/wordtap?", "charset=utf8mb4", "loc=UTC", "parseTime=True", "interpolateParams=true"},
		},
		{
			name:     "handles unescaped slash in timezone like loc=America/Lima and password question mark",
			input:    "user:p?ss@tcp(127.0.0.1:3306)/wordtap_db?charset=utf8mb4&parseTime=True&loc=America/Lima",
			contains: []string{"user:p?ss@tcp(127.0.0.1:3306)/wordtap_db?", "charset=utf8mb4", "loc=UTC", "parseTime=True", "interpolateParams=true"},
			excludes: []string{"America", "Lima"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeDSN(tc.input)
			for _, sub := range tc.contains {
				if !strings.Contains(got, sub) {
					t.Fatalf("expected DSN %q to contain %q", got, sub)
				}
			}
			for _, sub := range tc.excludes {
				if strings.Contains(got, sub) {
					t.Fatalf("expected DSN %q not to contain %q", got, sub)
				}
			}
			cfg, err := mysqlDriver.ParseDSN(got)
			if err != nil {
				t.Fatalf("mysqlDriver.ParseDSN(%q) failed: %v", got, err)
			}
			if cfg.Loc != time.UTC || !cfg.ParseTime {
				t.Fatalf("expected Loc=UTC and ParseTime=true, got Loc=%v ParseTime=%v", cfg.Loc, cfg.ParseTime)
			}
			if strings.HasPrefix(tc.input, "user:p?ss@") && cfg.Passwd != "p?ss" {
				t.Fatalf("expected password 'p?ss' intact, got %q", cfg.Passwd)
			}
		})
	}
}

func TestGetDSN(t *testing.T) {
	t.Run("fallback separate env vars uses loc=UTC and default port", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("DB_USER", "wordtap")
		t.Setenv("DB_PASSWORD", "secret")
		t.Setenv("DB_HOST", "127.0.0.1")
		t.Setenv("DB_PORT", "")
		t.Setenv("DB_NAME", "wordtap_db")

		dsn, err := getDSN()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, sub := range []string{"@tcp(127.0.0.1:3306)/wordtap_db?", "loc=UTC", "parseTime=True", "interpolateParams=true"} {
			if !strings.Contains(dsn, sub) {
				t.Fatalf("expected fallback DSN %q to contain %q", dsn, sub)
			}
		}
		if strings.Contains(dsn, "loc=Local") {
			t.Fatalf("expected fallback DSN %q not to contain loc=Local", dsn)
		}
	})

	t.Run("DATABASE_URL normalizes loc=UTC", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "root:pw@tcp(mysql:3306)/wt?loc=Local")

		dsn, err := getDSN()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(dsn, "loc=UTC") || !strings.Contains(dsn, "parseTime=True") {
			t.Fatalf("expected normalized DATABASE_URL %q to contain loc=UTC and parseTime=True", dsn)
		}
		if strings.Contains(dsn, "loc=Local") {
			t.Fatalf("expected normalized DATABASE_URL %q not to contain loc=Local", dsn)
		}
	})

	t.Run("missing env vars returns error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("DB_HOST", "")
		t.Setenv("DB_USER", "")
		t.Setenv("DB_NAME", "")

		if _, err := getDSN(); err == nil {
			t.Fatal("expected error when DB env vars are missing")
		}
	})
}
