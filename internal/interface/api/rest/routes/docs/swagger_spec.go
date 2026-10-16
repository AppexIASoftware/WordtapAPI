package docs

const SwaggerJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "WordtapAPI",
    "version": "1.0.0",
    "description": "Especificación oficial de la API REST de Wordtap. Arquitectura Hexagonal / Clean Architecture con patrón CQRS sobre Go Echo v5 y MySQL 8.0."
  },
  "servers": [
    {
      "url": "/",
      "description": "Servidor Actual"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "Token JWT emitido en /api/v1/auth/google. Ingrese solo el token sin la palabra Bearer."
      }
    }
  },
  "tags": [
    { "name": "Health", "description": "Monitoreo y diagnóstico de infraestructura" },
    { "name": "Auth", "description": "Autenticación e intercambio de tokens" },
    { "name": "Gamification", "description": "Gestión y calibración de actividades lúdicas docentes" },
    { "name": "Courses", "description": "Catálogo público y gestión docente de cursos" },
    { "name": "Lessons", "description": "Estructura de lecciones e ítems pedagógicos" },
    { "name": "Vault", "description": "Bóveda léxica oficial y arquetipos de vocabulario" },
    { "name": "Classroom", "description": "Gestión de cohortes, aulas e inscripciones" },
    { "name": "TeacherApplications", "description": "Postulaciones y revisión de docentes" },
    { "name": "Settings", "description": "Parámetros y toggles globales de la plataforma" }
  ],
  "paths": {
    "/health": {
      "get": {
        "tags": ["Health"],
        "summary": "Verificar salud del servicio y conexión a MySQL",
        "responses": {
          "200": { "description": "Servicio operativo" },
          "503": { "description": "Fallo en conexión de base de datos" }
        }
      }
    },
    "/api/v1/auth/google": {
      "post": {
        "tags": ["Auth"],
        "summary": "Intercambiar Google ID Token por sesión JWT",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["id_token"],
                "properties": {
                  "id_token": { "type": "string" },
                  "device_info": { "type": "object" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Sesión iniciada con éxito (retorna tokens y perfil)" },
          "401": { "description": "Token inválido o expirado" }
        }
      }
    },
    "/api/v1/teacher/games": {
      "get": {
        "tags": ["Gamification"],
        "summary": "Listar actividades personalizadas creadas por el docente autenticado",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Lista de actividades en formato JSON" },
          "401": { "description": "No autorizado" }
        }
      },
      "post": {
        "tags": ["Gamification"],
        "summary": "Crear una nueva actividad lúdica personalizada en MySQL",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["base_archetype", "title", "level", "time_limit_seconds", "allowed_mistakes", "reward_xp", "item_count"],
                "properties": {
                  "course_id": { "type": "string", "nullable": true },
                  "base_archetype": { "type": "string", "example": "match" },
                  "title": { "type": "string", "example": "Desafío de Verbos B1" },
                  "description": { "type": "string" },
                  "level": { "type": "string", "example": "B1 Intermedio" },
                  "time_limit_seconds": { "type": "integer", "example": 45 },
                  "allowed_mistakes": { "type": "integer", "example": 3 },
                  "reward_xp": { "type": "integer", "example": 20 },
                  "item_count": { "type": "integer", "example": 6 },
                  "source_type": { "type": "string", "example": "vault" },
                  "custom_words_json": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Actividad creada con éxito" },
          "400": { "description": "Datos de entrada inválidos" }
        }
      }
    },
    "/api/v1/teacher/games/{id}": {
      "get": {
        "tags": ["Gamification"],
        "summary": "Obtener detalle de una actividad lúdica por ID",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "Detalle de la actividad" },
          "404": { "description": "Actividad no encontrada" }
        }
      },
      "put": {
        "tags": ["Gamification"],
        "summary": "Actualizar una actividad lúdica existente",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "Actividad actualizada" },
          "404": { "description": "Actividad no encontrada" }
        }
      },
      "delete": {
        "tags": ["Gamification"],
        "summary": "Eliminar una actividad lúdica de MySQL",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "Actividad eliminada correctamente" },
          "404": { "description": "No encontrada" }
        }
      }
    },
    "/api/v1/courses": {
      "get": {
        "tags": ["Courses"],
        "summary": "Catálogo general de cursos aprobados y publicados (Admin / Moderador)",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Lista de cursos" },
          "403": { "description": "Prohibido para docentes (deben usar /teacher/courses)" }
        }
      }
    },
    "/api/v1/teacher/courses": {
      "get": {
        "tags": ["Courses"],
        "summary": "Listar cursos creados por el docente autenticado",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Cursos del docente" }
        }
      },
      "post": {
        "tags": ["Courses"],
        "summary": "Crear un nuevo borrador de curso",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "201": { "description": "Curso creado" }
        }
      }
    },
    "/api/v1/vault/items": {
      "get": {
        "tags": ["Vault"],
        "summary": "Explorador de ítems pedagógicos certificados de la Bóveda",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Ítems de vocabulario, frases y arquetipos" }
        }
      }
    },
    "/api/v1/classroom": {
      "get": {
        "tags": ["Classroom"],
        "summary": "Listar aulas y cohortes del docente",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Lista de aulas activas" }
        }
      }
    },
    "/api/v1/settings/public": {
      "get": {
        "tags": ["Settings"],
        "summary": "Consultar configuraciones públicas de la plataforma",
        "responses": {
          "200": { "description": "Toggles operativos globales" }
        }
      }
    }
  }
}
`
