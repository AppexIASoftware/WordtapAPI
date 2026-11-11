package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/push"
	"github.com/google/uuid"
)

var (
	ErrEmptyVaultItemID   = errors.New("el id del item de vocabulario es requerido")
	ErrVaultItemNotFound  = errors.New("item de vocabulario no encontrado")
	ErrClassroomForbidden = errors.New("no tienes permisos sobre esta aula")
)

type DispatchVaultWordCommand struct {
	ItemVaultID       string            `json:"item_vault_id"`
	TargetAudience    string            `json:"target_audience"` // "all", "cefr", "classroom"
	TargetClassroomID *string           `json:"target_classroom_id,omitempty"`
	TargetCEFRLevel   *string           `json:"target_cefr_level,omitempty"`
	SenderUserID      string            `json:"sender_user_id"`
	SenderRole        entities.UserRole `json:"sender_role"`
}

type DispatchVaultWordResult struct {
	Word            string `json:"word"`
	Translation     string `json:"translation"`
	RecipientsCount int    `json:"recipients_count"`
	DevicesCount    int    `json:"devices_count"`
}

type DispatchVaultWordHandler struct {
	bankRepo      repositories.ContentBankRepository
	classroomRepo repositories.ClassroomRepository
	userRepo      repositories.UserRepository
	inboxRepo     inbox.Repository
	deviceRepo    repositories.DeviceRepository
	pushSender    push.PushSender
}

func NewDispatchVaultWordHandler(
	bankRepo repositories.ContentBankRepository,
	classroomRepo repositories.ClassroomRepository,
	userRepo repositories.UserRepository,
	inboxRepo inbox.Repository,
	deviceRepo repositories.DeviceRepository,
	pushSender push.PushSender,
) *DispatchVaultWordHandler {
	return &DispatchVaultWordHandler{
		bankRepo:      bankRepo,
		classroomRepo: classroomRepo,
		userRepo:      userRepo,
		inboxRepo:     inboxRepo,
		deviceRepo:    deviceRepo,
		pushSender:    pushSender,
	}
}

func (h *DispatchVaultWordHandler) Handle(ctx context.Context, cmd DispatchVaultWordCommand) (*DispatchVaultWordResult, error) {
	cmd.ItemVaultID = strings.TrimSpace(cmd.ItemVaultID)
	if cmd.ItemVaultID == "" {
		return nil, ErrEmptyVaultItemID
	}

	item, err := h.bankRepo.FindItemByID(ctx, cmd.ItemVaultID)
	if err != nil || item == nil {
		return nil, ErrVaultItemNotFound
	}

	var userIDs []string
	switch strings.ToLower(strings.TrimSpace(cmd.TargetAudience)) {
	case "classroom":
		if cmd.TargetClassroomID == nil || strings.TrimSpace(*cmd.TargetClassroomID) == "" {
			return nil, errors.New("id del aula requerido para envío por aula")
		}
		clsID := strings.TrimSpace(*cmd.TargetClassroomID)
		cls, err := h.classroomRepo.FindByID(ctx, clsID)
		if err != nil || cls == nil {
			return nil, errors.New("aula no encontrada")
		}
		if cmd.SenderRole == entities.RoleInstructor && cls.TeacherID != cmd.SenderUserID {
			return nil, ErrClassroomForbidden
		}
		enrollments, err := h.classroomRepo.ListEnrollments(ctx, clsID)
		if err != nil {
			return nil, fmt.Errorf("failed to list enrollments: %w", err)
		}
		for _, en := range enrollments {
			if en.StudentID != "" {
				userIDs = append(userIDs, en.StudentID)
			}
		}

	case "cefr":
		level := item.CEFRLevel
		if cmd.TargetCEFRLevel != nil && strings.TrimSpace(*cmd.TargetCEFRLevel) != "" {
			level = strings.ToUpper(strings.TrimSpace(*cmd.TargetCEFRLevel))
		}
		studentRole := entities.RoleStudent
		uIDs, err := h.userRepo.ListUserIDs(ctx, &studentRole, &level)
		if err != nil {
			return nil, fmt.Errorf("failed to list users by cefr: %w", err)
		}
		userIDs = uIDs

	default: // "all"
		studentRole := entities.RoleStudent
		uIDs, err := h.userRepo.ListUserIDs(ctx, &studentRole, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}
		userIDs = uIDs
	}

	if len(userIDs) == 0 {
		return &DispatchVaultWordResult{
			Word:            item.TargetText,
			Translation:     item.PromptText,
			RecipientsCount: 0,
			DevicesCount:    0,
		}, nil
	}

	title := fmt.Sprintf("Reto de vocabulario: %s", item.TargetText)
	body := item.PromptText
	if item.IPA != nil && *item.IPA != "" {
		body = fmt.Sprintf("%s • %s", item.PromptText, *item.IPA)
	}
	actionURL := fmt.Sprintf("/vault/%s", item.ID)

	now := time.Now().UTC()
	notifications := make([]entities.Notification, len(userIDs))
	for i, uID := range userIDs {
		notifications[i] = entities.Notification{
			ID:               uuid.NewString(),
			UserID:           uID,
			NotificationType: entities.NotificationTypeDailyWord,
			Title:            title,
			Body:             body,
			ActionURL:        &actionURL,
			SentAt:           &now,
			CreatedAt:        now,
		}
	}

	if err := h.inboxRepo.CreateBatch(ctx, notifications); err != nil {
		return nil, fmt.Errorf("failed to save notifications in inbox: %w", err)
	}

	tokens, err := h.deviceRepo.GetTokensByUsers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user tokens: %w", err)
	}

	devicesDispatched := 0
	if h.pushSender != nil && len(tokens) > 0 {
		data := map[string]string{
			"type":        "vault_challenge",
			"item_id":     item.ID,
			"word":        item.TargetText,
			"translation": item.PromptText,
			"action_url":  actionURL,
		}

		for _, token := range tokens {
			if sendErr := h.pushSender.Send(ctx, token, title, body, data); sendErr == nil {
				devicesDispatched++
			}
		}
	}

	return &DispatchVaultWordResult{
		Word:            item.TargetText,
		Translation:     item.PromptText,
		RecipientsCount: len(userIDs),
		DevicesCount:    devicesDispatched,
	}, nil
}
