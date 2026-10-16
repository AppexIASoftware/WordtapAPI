package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type CreateVaultItemCommand struct {
	BankType    string
	PromptText  string
	TargetText  string
	IPA         *string
	CEFRLevel   string
	Difficulty  int
	AudioURL    *string
	Explanation *string
	Metadata    string
	IsOfficial  bool
	RequesterID string
	Role        entities.UserRole
}

type CreateVaultItemHandler struct {
	repo repositories.ContentBankRepository
}

func NewCreateVaultItemHandler(repo repositories.ContentBankRepository) *CreateVaultItemHandler {
	return &CreateVaultItemHandler{repo: repo}
}

func (h *CreateVaultItemHandler) Handle(ctx context.Context, cmd CreateVaultItemCommand) (*entities.ContentBankItem, error) {
	prompt := strings.TrimSpace(cmd.PromptText)
	target := strings.TrimSpace(cmd.TargetText)
	bankType := strings.TrimSpace(cmd.BankType)

	if prompt == "" || target == "" || bankType == "" {
		return nil, apperrors.New("Completa el texto de origen, el texto de destino y el tipo de contenido.")
	}

	cefr := strings.ToUpper(strings.TrimSpace(cmd.CEFRLevel))
	if cefr == "" {
		cefr = "A1"
	}

	diff := cmd.Difficulty
	if diff < 1 || diff > 3 {
		diff = 1
	}

	meta := cmd.Metadata
	if strings.TrimSpace(meta) == "" {
		meta = "{}"
	}

	var createdBy *string
	status := entities.ContentStatusPublished

	if cmd.IsOfficial {
		if cmd.Role != entities.RoleAdmin && cmd.Role != entities.RoleModerator {
			return nil, apperrors.New("Solo administradores y moderadores pueden crear ítems oficiales.")
		}
		createdBy = nil
	} else {
		if cmd.RequesterID == "" {
			return nil, errors.New("unauthorized: missing requester id for private vault item")
		}
		reqID := cmd.RequesterID
		createdBy = &reqID
	}

	item := &entities.ContentBankItem{
		ID:          uuid.NewString(),
		BankType:    bankType,
		SourceLang:  "es",
		TargetLang:  "en",
		PromptText:  prompt,
		TargetText:  target,
		IPA:         cmd.IPA,
		CEFRLevel:   cefr,
		Difficulty:  diff,
		AudioURL:    cmd.AudioURL,
		Explanation: cmd.Explanation,
		Metadata:    meta,
		Status:      status,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}
