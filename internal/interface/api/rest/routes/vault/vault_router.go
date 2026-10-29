package vault

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type VaultRouter struct {
	listTypesHandler      *queries.ListVaultTypesHandler
	listCategoriesHandler *queries.ListVaultCategoriesHandler
	createCategoryHandler *commands.CreateVaultCategoryHandler
	listItemsHandler      *queries.ListVaultItemsHandler
	createItemHandler     *commands.CreateVaultItemHandler
	updateItemHandler     *commands.UpdateVaultItemHandler
	deleteItemHandler     *commands.DeleteVaultItemHandler
}

func NewVaultRouter(
	listTypesHandler *queries.ListVaultTypesHandler,
	listCategoriesHandler *queries.ListVaultCategoriesHandler,
	createCategoryHandler *commands.CreateVaultCategoryHandler,
	listItemsHandler *queries.ListVaultItemsHandler,
	createItemHandler *commands.CreateVaultItemHandler,
	updateItemHandler *commands.UpdateVaultItemHandler,
	deleteItemHandler *commands.DeleteVaultItemHandler,
) *VaultRouter {
	return &VaultRouter{
		listTypesHandler:      listTypesHandler,
		listCategoriesHandler: listCategoriesHandler,
		createCategoryHandler: createCategoryHandler,
		listItemsHandler:      listItemsHandler,
		createItemHandler:     createItemHandler,
		updateItemHandler:     updateItemHandler,
		deleteItemHandler:     deleteItemHandler,
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
	vaultGroup.GET("/categories", r.ListCategories, staffRoles)
	vaultGroup.POST("/categories", r.CreateCategory, adminOrModerator)
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
	CategoryID  *string `json:"category_id"`
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
	BankType    *string `json:"bank_type"`
	CategoryID  entities.OptionalString `json:"category_id"`
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
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar los tipos de contenido.", err)
	}
	return c.JSON(http.StatusOK, types)
}

func (r *VaultRouter) ListCategories(c *echo.Context) error {
	categories, err := r.listCategoriesHandler.Handle(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar las categorías de contenido.", err)
	}
	return c.JSON(http.StatusOK, categories)
}

type CreateVaultCategoryRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (r *VaultRouter) CreateCategory(c *echo.Context) error {
	var req CreateVaultCategoryRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Datos de categoría inválidos.", err)
	}

	category, err := r.createCategoryHandler.Handle(c.Request().Context(), commands.CreateVaultCategoryCommand{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo crear la categoría.", err)
	}

	return c.JSON(http.StatusCreated, category)
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

	var categoryIDPtr *string
	if cat := c.QueryParam("category_id"); cat != "" && cat != "all" {
		categoryIDPtr = &cat
	} else if catSlug := c.QueryParam("category"); catSlug != "" && catSlug != "all" {
		categoryIDPtr = &catSlug
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
		CategoryID:  categoryIDPtr,
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
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar los ítems de la bóveda.", err)
	}
	return c.JSON(http.StatusOK, res)
}

func (r *VaultRouter) CreateOfficialItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	var req CreateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	item, err := r.createItemHandler.Handle(c.Request().Context(), commands.CreateVaultItemCommand{
		BankType:    req.BankType,
		CategoryID:  req.CategoryID,
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
		return response.Error(c, http.StatusBadRequest, "No se pudo crear el ítem.", err)
	}
	return c.JSON(http.StatusCreated, item)
}

func (r *VaultRouter) UpdateOfficialItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	id := c.Param("id")

	var req UpdateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	item, err := r.updateItemHandler.Handle(c.Request().Context(), commands.UpdateVaultItemCommand{
		ID:          id,
		BankType:    req.BankType,
		CategoryID:  req.CategoryID.Value,
		CategoryIDSet: req.CategoryID.Set,
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
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar el ítem.", err)
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
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar el ítem.", err)
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

	var categoryIDPtr *string
	if cat := c.QueryParam("category_id"); cat != "" && cat != "all" {
		categoryIDPtr = &cat
	} else if catSlug := c.QueryParam("category"); catSlug != "" && catSlug != "all" {
		categoryIDPtr = &catSlug
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
		CategoryID:  categoryIDPtr,
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
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar tus ítems.", err)
	}
	return c.JSON(http.StatusOK, res)
}

func (r *VaultRouter) CreateTeacherPrivateItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	var req CreateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	item, err := r.createItemHandler.Handle(c.Request().Context(), commands.CreateVaultItemCommand{
		BankType:    req.BankType,
		CategoryID:  req.CategoryID,
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
		return response.Error(c, http.StatusBadRequest, "No se pudo crear el ítem.", err)
	}
	return c.JSON(http.StatusCreated, item)
}

func (r *VaultRouter) UpdateTeacherPrivateItem(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	id := c.Param("id")

	var req UpdateVaultItemRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	item, err := r.updateItemHandler.Handle(c.Request().Context(), commands.UpdateVaultItemCommand{
		ID:          id,
		BankType:    req.BankType,
		CategoryID:  req.CategoryID.Value,
		CategoryIDSet: req.CategoryID.Set,
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
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar el ítem.", err)
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
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar el ítem.", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"deleted": true, "id": id})
}
