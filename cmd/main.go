package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/db/mysql"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest"
)

func main() {
	// 1. Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, reading environment variables from system")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Conectar a MySQL
	db, err := mysql.NewConnection()
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}

	// 3. Inicializar y encender el servidor REST
	server := rest.NewServer(db)

	log.Printf("Server listening on port :%s", port)
	if err := server.Start(":" + port); err != nil {
		log.Fatalf("Server shutdown: %v", err)
	}
}
