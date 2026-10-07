package mysql

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func getDSN() (string, error) {
	// Si existe una URL completa, la usa asegurando parámetros de alto rendimiento
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		if !strings.Contains(dsn, "interpolateParams=") {
			if strings.Contains(dsn, "?") {
				dsn += "&interpolateParams=true"
			} else {
				dsn += "?interpolateParams=true"
			}
		}
		return dsn, nil
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
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true&timeout=5s&readTimeout=10s&writeTimeout=10s",
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
