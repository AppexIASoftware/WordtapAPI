package notificationdispatch

import (
	"context"
	"fmt"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/push"
)

type DispatchDailyWordCommand struct {
	UserID string
}

type DispatchDailyWordResult struct {
	Dispatched   bool   `json:"dispatched"`
	Word         string `json:"word,omitempty"`
	Translation  string `json:"translation,omitempty"`
	DevicesCount int    `json:"devices_count"`
}

type DispatchDailyWordHandler struct {
	bankRepo   domainRepo.ContentBankRepository
	inboxRepo  inbox.Repository
	deviceRepo domainRepo.DeviceRepository
	pushSender push.PushSender
}

func NewDispatchDailyWordHandler(
	bankRepo domainRepo.ContentBankRepository,
	inboxRepo inbox.Repository,
	deviceRepo domainRepo.DeviceRepository,
	pushSender push.PushSender,
) *DispatchDailyWordHandler {
	return &DispatchDailyWordHandler{
		bankRepo:   bankRepo,
		inboxRepo:  inboxRepo,
		deviceRepo: deviceRepo,
		pushSender: pushSender,
	}
}

func (h *DispatchDailyWordHandler) Handle(ctx context.Context, cmd DispatchDailyWordCommand) (*DispatchDailyWordResult, error) {
	if cmd.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	published := entities.ContentStatusPublished
	filter := domainRepo.ContentBankFilter{
		CreatedBy: nil,
		OnlyMine:  false,
		Status:    &published,
		Limit:     1,
		Offset:    0,
	}

	items, total, err := h.bankRepo.ListItems(ctx, filter)
	if err != nil || total == 0 || len(items) == 0 {
		return &DispatchDailyWordResult{Dispatched: false}, err
	}

	item := items[0]
	now := time.Now().UTC()

	title := fmt.Sprintf("Palabra del día: %s", item.TargetText)
	body := item.PromptText
	if item.IPA != nil && *item.IPA != "" {
		body = fmt.Sprintf("%s • %s", item.PromptText, *item.IPA)
	}

	notification := &entities.Notification{
		UserID:           cmd.UserID,
		NotificationType: entities.NotificationTypeDailyWord,
		Title:            title,
		Body:             body,
		SentAt:           &now,
		CreatedAt:        now,
	}

	if err := h.inboxRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to save notification in inbox: %w", err)
	}

	tokens, _ := h.deviceRepo.GetTokensByUser(ctx, cmd.UserID)
	dispatchedCount := 0

	if h.pushSender != nil && len(tokens) > 0 {
		data := map[string]string{
			"type":            "daily_word",
			"notification_id": notification.ID,
			"word":            item.TargetText,
			"translation":     item.PromptText,
		}
		for _, token := range tokens {
			if sendErr := h.pushSender.Send(ctx, token, title, body, data); sendErr == nil {
				dispatchedCount++
			}
		}
	}

	return &DispatchDailyWordResult{
		Dispatched:   true,
		Word:         item.TargetText,
		Translation:  item.PromptText,
		DevicesCount: dispatchedCount,
	}, nil
}
