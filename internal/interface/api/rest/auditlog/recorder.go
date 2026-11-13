package auditlog

import (
	"encoding/json"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// Record persists an admin audit log entry when an AuditRepository is attached.
func Record(c *echo.Context, repo domainRepo.AuditRepository, action, entityType, entityID string, payload map[string]any) {
	if repo == nil || c == nil {
		return
	}
	actorID, _ := c.Get("user_id").(string)
	ip := c.RealIP()
	raw, _ := json.Marshal(payload)

	var actorPtr, idPtr, ipPtr *string
	if strings.TrimSpace(actorID) != "" {
		actorPtr = &actorID
	}
	if strings.TrimSpace(entityID) != "" {
		idPtr = &entityID
	}
	if strings.TrimSpace(ip) != "" {
		ipPtr = &ip
	}

	_ = repo.CreateAuditLog(c.Request().Context(), &entities.AuditLog{
		ActorUserID: actorPtr,
		Action:      action,
		EntityType:  entityType,
		EntityID:    idPtr,
		Changes:     string(raw),
		IPAddress:   ipPtr,
	})
}
