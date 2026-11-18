package mysql

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func normalizeDSN(raw string) string {
	dsn := strings.TrimSpace(raw)
	if dsn == "" {
		return ""
	}
	// Split query string at the /dbname? boundary so "?" in passwords or "/" in query values (e.g. loc=America/Lima) stays intact.
	dbSlashIdx := -1
	if addrEnd := strings.Index(dsn, ")/"); addrEnd >= 0 {
		dbSlashIdx = addrEnd + 1
	} else if atSlash := strings.LastIndex(dsn, "@/"); atSlash >= 0 {
		dbSlashIdx = atSlash + 1
	} else {
		dbSlashIdx = strings.Index(dsn, "/")
	}
	queryStart := -1
	if dbSlashIdx >= 0 {
		if qOffset := strings.Index(dsn[dbSlashIdx+1:], "?"); qOffset >= 0 {
			queryStart = dbSlashIdx + 1 + qOffset
		}
	} else if qOffset := strings.Index(dsn, "?"); qOffset >= 0 {
		queryStart = qOffset
	}
	if queryStart < 0 {
		return dsn + "?charset=utf8mb4&parseTime=True&loc=UTC&interpolateParams=true"
	}

	base, rawQuery := dsn[:queryStart], dsn[queryStart+1:]
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		parts := strings.Split(rawQuery, "&")
		kept := make([]string, 0, len(parts)+4)
		hasCharset := false
		hasInterpolate := false
		for _, part := range parts {
			if part == "" {
				continue
			}
			lower := strings.ToLower(part)
			if strings.HasPrefix(lower, "loc=") || strings.HasPrefix(lower, "parsetime=") {
				continue
			}
			if strings.HasPrefix(lower, "charset=") {
				hasCharset = true
			} else if strings.HasPrefix(lower, "interpolateparams=") {
				hasInterpolate = true
			}
			kept = append(kept, part)
		}
		if !hasCharset {
			kept = append(kept, "charset=utf8mb4")
		}
		if !hasInterpolate {
			kept = append(kept, "interpolateParams=true")
		}
		kept = append(kept, "parseTime=True", "loc=UTC")
		return base + "?" + strings.Join(kept, "&")
	}

	if q.Get("charset") == "" {
		q.Set("charset", "utf8mb4")
	}
	q.Set("parseTime", "True")
	q.Set("loc", "UTC")
	if q.Get("interpolateParams") == "" {
		q.Set("interpolateParams", "true")
	}
	return base + "?" + q.Encode()
}

func getDSN() (string, error) {
	// Si existe una URL completa, la usa asegurando parámetros de alto rendimiento y zona horaria UTC
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		return normalizeDSN(dsn), nil
	}

	// Variables separadas requeridas
	user := strings.TrimSpace(os.Getenv("DB_USER"))
	password := os.Getenv("DB_PASSWORD")
	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))

	if host == "" || user == "" || dbName == "" {
		return "", fmt.Errorf("missing required database environment variables: DB_HOST, DB_USER, DB_NAME (or DATABASE_URL)")
	}
	if port == "" {
		port = "3306"
	}

	// Formato optimizado: interpolateParams=true reduce roundtrips de red en ~66% al omitir COM_STMT_PREPARE/CLOSE individuales
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC&interpolateParams=true&timeout=5s&readTimeout=10s&writeTimeout=10s",
		user, password, host, port, dbName,
	), nil
}

func NewConnection() (*gorm.DB, error) {
	dsn, err := getDSN()
	if err != nil {
		return nil, err
	}

	fmt.Println("Connecting to MySQL with high-performance configuration...")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt:            true, // Caché interno de sentencias preparadas en GORM
		SkipDefaultTransaction: true, // Omite transacciones automáticas en operaciones individuales (60% más rápido)
		Logger:                 logger.Default.LogMode(logger.Warn), // Silencia logging I/O síncrono que frena peticiones
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Pool de conexiones nativo optimizado para baja latencia
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get generic database object: %w", err)
	}

	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)
	sqlDB.SetConnMaxLifetime(15 * time.Minute)

	// Verificación de conectividad inmediata
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	fmt.Println("MySQL connected successfully with low-latency pool!")
	return db, nil
}
