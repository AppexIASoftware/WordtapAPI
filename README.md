# WordtapAPI

Servicio backend de API REST para la plataforma móvil de aprendizaje de idiomas **WordTap**.

## Descripción general

WordtapAPI gestiona usuarios, progreso de aprendizaje, sesiones de minijuegos interactivos, notificaciones y procesamiento de pagos o suscripciones. Está desarrollado bajo los principios de **Clean Architecture** (Arquitectura Limpia) y el patrón **CQRS (Command Query Responsibility Segregation)** en Go, utilizando **Echo v5** para el enrutamiento HTTP y **GORM** para la persistencia en MySQL.

---

## Arquitectura y estructura del proyecto

El código fuente está organizado en cuatro capas desacopladas:

```text
WordtapAPI/
├── cmd/
│   └── main.go                                  # Punto de entrada de la aplicación
├── internal/
│   ├── domain/                                  # Modelos de negocio puros y contratos
│   │   ├── entities/                            # Entidades del dominio
│   │   └── repositories/                        # Interfaces abstractas de repositorios
│   │
│   ├── application/                             # Casos de uso del negocio (CQRS)
│   │   └── features/
│   │       ├── User/                            # Dominio de usuario (comandos y consultas)
│   │       ├── Lesson/                          # Lecciones y progreso de vocabulario
│   │       ├── Game/                            # Minijuegos, niveles y puntaje de estrellas
│   │       ├── Notification/                    # Configuración e historial de notificaciones
│   │       └── Payment/                         # Compras de cursos y suscripciones
│   │
│   ├── infrastructure/                          # Adaptadores secundarios y persistencia
│   │   ├── db/mysql/                            # Conexión GORM y configuración de pool
│   │   └── repositories/                        # Implementaciones concretas de repositorios (MySQL)
│   │
│   └── interface/                               # Adaptadores primarios (HTTP REST)
│       └── api/
│           └── rest/
│               ├── middlewares/                 # Autenticación, registro, recuperación y CORS
│               ├── routes/                      # Controladores agrupados por característica
│               └── server.go                    # Configuración del servidor Echo
├── .env.example
├── .gitignore
├── go.mod
└── go.sum
```

---

## Tecnologías utilizadas

- **Lenguaje:** Go (1.25+)
- **Framework HTTP:** Echo v5 (`github.com/labstack/echo/v5`)
- **ORM / Acceso a datos:** GORM (`gorm.io/gorm`)
- **Driver de base de datos:** MySQL Driver (`gorm.io/driver/mysql`)
- **Gestión de entorno:** Godotenv (`github.com/joho/godotenv`)

---

## Puesta en marcha

### Requisitos previos

- Go `1.25` o superior instalado.
- Instancia de MySQL `8.0+` ejecutándose localmente o en la nube.

### Configuración del entorno

Crear un archivo `.env` en la raíz del proyecto basándose en `.env.example`:

```env
# Servidor HTTP
PORT=8080

# Base de datos MySQL
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=usuario
DB_PASSWORD=password
DB_NAME=wordtap_db

# Control de migraciones y datos iniciales (opcional)
AUTO_MIGRATE=true
AUTO_SEED=true
```

### Ejecución de la API

1. **Instalar y sincronizar dependencias:**
   ```bash
   go mod tidy
   ```

2. **Ejecutar en modo desarrollo:**
   ```bash
   go run cmd/main.go
   ```

3. **Compilar binario ejecutable:**
   ```bash
   go build -o tmp/main.exe cmd/main.go
   ./tmp/main.exe
   ```

---

## Diagnóstico y estado del servicio

- **Endpoint de verificación de salud:**
  ```http
  GET /api/services/v1/health
  ```
  Retorna el estado operativo del servidor, la verificación de conexión activa con la base de datos MySQL y el tiempo de respuesta (ping) en milisegundos.
