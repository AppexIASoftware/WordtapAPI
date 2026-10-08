package vault

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type VaultRouter struct {
	listTypesHandler  *queries.ListVaultTypesHandler
	listItemsHandler  *queries.ListVaultItemsHandler
	createItemHandler *commands.CreateVaultItemHandler
	updateItemHandler *commands.UpdateVaultItemHandler
	deleteItemHandler *commands.DeleteVaultItemHandler
}

func NewVaultRouter(
	listTypesHandler *queries.ListVaultTypesHandler,
	listItemsHandler *queries.ListVaultItemsHandler,
	createItemHandler *commands.CreateVaultItemHandler,
	updateItemHandler *commands.UpdateVaultItemHandler,
	deleteItemHandler *commands.DeleteVaultItemHandler,
) *VaultRouter {
	return &VaultRouter{
		listTypesHandler:  listTypesHandler,
		listItemsHandler:  listItemsHandler,
		createItemHandler: createItemHandler,
		updateItemHandler: updateItemHandler,
		deleteItemHandler: deleteItemHandler,
	}
}

func (r *VaultRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	staffRoles echo.MiddlewareFunc,
	adminOrModerator echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	// 1. Banco Oficial Wordtap (Público / Curado por lingüistas)
	vaultGroup := v1.Group("/vault")
	vaultGroup.Use(authRequired)
	vaultGroup.GET("/types", r.ListTypes, staffRoles)
	vaultGroup.GET("/items", r.ListOfficialItems, staffRoles)
	vaultGroup.POST("/items", r.CreateOfficialItem, adminOrModerator)
	vaultGroup.PUT("/items/:id", r.UpdateOfficialItem, adminOrModerator)
	vaultGroup.DELETE("/items/:id", r.DeleteOfficialItem, adminOrModerator)

	// 2. Biblioteca Privada del Docente (Aislada por usuario)
	teacherVaultGroup := v1.Group("/teacher/vault")
	teacherVaultGroup.Use(authRequired, instructorOrAdmin)
	teacherVaultGroup.GET("/items", r.ListTeacherPrivateItems)
	teacherVaultGroup.POST("/items", r.CreateTeacherPrivateItem)
	teacherVaultGroup.PUT("/items/:id", r.UpdateTeacherPrivateItem)
	teacherVaultGroup.DELETE("/items/:id", r.DeleteTeacherPrivateItem)
}

type CreateVaultItemRequest struct {
	BankType    string  `json:"bank_type"`
	PromptText  string  `json:"prompt_text"`
	TargetText  string  `json:"target_text"`
	IPA         *string `json:"ipa"`
	CEFRLevel   string  `json:"cefr_level"`
	Difficulty  int     `json:"difficulty"`
	AudioURL    *string `json:"audio_url"`
	Explanation *string `json:"explanation"`
	Metadata    string  `json:"metadata"`
}

type UpdateVaultItemRequest struct {
	PromptText  *string `json:"prompt_text"`
	TargetText  *string `json:"target_text"`
	IPA         *string `json:"ipa"`
	CEFRLevel   *string `json:"cefr_level"`
	Difficulty  *int    `json:"difficulty"`
	AudioURL    *string `json:"audio_url"`
	Explanation *string `json:"explanation"`
	Metadata    *string `json:"metadata"`
}

func (r *VaultRouter) ListTypes(c *echo.Context) error {
	types, err := r.listTypesHandler.Handle(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to list vault types"})
	}
	return c.JSON(http.StatusOK, types)
}

func (r *VaultRouter) ListOfficialItems(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	var diffPtr *int
	if diffStr := c.QueryParam("difficulty"); diffStr != "" && diffStr != "all" {
		if d, err := strconv.Atoi(diffStr); err == nil && d > 0 {
			diffPtr = &d
		}
	}

	var bankTypePtr *string
	if bt := c.QueryParam("bank_type"); bt != "" && bt != "all" {
		bankTypePtr = &bt
	}

	var cefrPtr *string
	if cefr := c.QueryParam("cefr_level"); cefr != "" && cefr != "all" {
		cefrPtr = &cefr
	}

	var searchPtr *string
	if q := c.QueryParam("q"); q != "" {
		searchPtr = &q
	}

	res, err := r.listItemsHandler.Handle(c.Request().Context(), queries.ListVaultItemsQuery{
		BankType:    bankTypePtr,
		CEFRLevel:   cefrPtr,
		Difficulty:  diffPtr,
		Search:      searchPtr,
		OnlyMine:    false,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

func (r *VaultRouter) CreateOfficialItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	var req CreateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	item, err := r.createItemHandler.Handle(c.Request().Context(), commands.CreateVaultItemCommand{
		BankType:    req.BankType,
		PromptText:  req.PromptText,
		TargetText:  req.TargetText,
		IPA:         req.IPA,
		CEFRLevel:   req.CEFRLevel,
		Difficulty:  req.Difficulty,
		AudioURL:    req.AudioURL,
		Explanation: req.Explanation,
		Metadata:    req.Metadata,
		IsOfficial:  true,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, item)
}

func (r *VaultRouter) UpdateOfficialItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	id := c.Param("id")

	var req UpdateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	item, err := r.updateItemHandler.Handle(c.Request().Context(), commands.UpdateVaultItemCommand{
		ID:          id,
		PromptText:  req.PromptText,
		TargetText:  req.TargetText,
		IPA:         req.IPA,
		CEFRLevel:   req.CEFRLevel,
		Difficulty:  req.Difficulty,
		AudioURL:    req.AudioURL,
		Explanation: req.Explanation,
		Metadata:    req.Metadata,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, item)
}

func (r *VaultRouter) DeleteOfficialItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	id := c.Param("id")

	err := r.deleteItemHandler.Handle(c.Request().Context(), commands.DeleteVaultItemCommand{
		ID:          id,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"deleted": true, "id": id})
}

func (r *VaultRouter) ListTeacherPrivateItems(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	var diffPtr *int
	if diffStr := c.QueryParam("difficulty"); diffStr != "" && diffStr != "all" {
		if d, err := strconv.Atoi(diffStr); err == nil && d > 0 {
			diffPtr = &d
		}
	}

	var bankTypePtr *string
	if bt := c.QueryParam("bank_type"); bt != "" && bt != "all" {
		bankTypePtr = &bt
	}

	var cefrPtr *string
	if cefr := c.QueryParam("cefr_level"); cefr != "" && cefr != "all" {
		cefrPtr = &cefr
	}

	var searchPtr *string
	if q := c.QueryParam("q"); q != "" {
		searchPtr = &q
	}

	res, err := r.listItemsHandler.Handle(c.Request().Context(), queries.ListVaultItemsQuery{
		BankType:    bankTypePtr,
		CEFRLevel:   cefrPtr,
		Difficulty:  diffPtr,
		Search:      searchPtr,
		OnlyMine:    true,
		TeacherID:   userID,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

func (r *VaultRouter) CreateTeacherPrivateItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	var req CreateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	item, err := r.createItemHandler.Handle(c.Request().Context(), commands.CreateVaultItemCommand{
		BankType:    req.BankType,
		PromptText:  req.PromptText,
		TargetText:  req.TargetText,
		IPA:         req.IPA,
		CEFRLevel:   req.CEFRLevel,
		Difficulty:  req.Difficulty,
		AudioURL:    req.AudioURL,
		Explanation: req.Explanation,
		Metadata:    req.Metadata,
		IsOfficial:  false,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, item)
}

func (r *VaultRouter) UpdateTeacherPrivateItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	id := c.Param("id")

	var req UpdateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}

	item, err := r.updateItemHandler.Handle(c.Request().Context(), commands.UpdateVaultItemCommand{
		ID:          id,
		PromptText:  req.PromptText,
		TargetText:  req.TargetText,
		IPA:         req.IPA,
		CEFRLevel:   req.CEFRLevel,
		Difficulty:  req.Difficulty,
		AudioURL:    req.AudioURL,
		Explanation: req.Explanation,
		Metadata:    req.Metadata,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, item)
}

func (r *VaultRouter) DeleteTeacherPrivateItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	id := c.Param("id")

	err := r.deleteItemHandler.Handle(c.Request().Context(), commands.DeleteVaultItemCommand{
		ID:          id,
		RequesterID: userID,
		Role:        entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"deleted": true, "id": id})
}
