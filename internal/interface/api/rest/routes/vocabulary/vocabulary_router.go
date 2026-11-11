package vocabulary

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vocabulary/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type VocabularyRouter struct {
	searchHandler *queries.SearchVocabularyHandler
}

func NewVocabularyRouter(searchHandler *queries.SearchVocabularyHandler) *VocabularyRouter {
	return &VocabularyRouter{
		searchHandler: searchHandler,
	}
}

func (r *VocabularyRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	vocab := v1.Group("/vocabulary")
	vocab.Use(authRequired)
	vocab.GET("/search", r.Search)
}

func (r *VocabularyRouter) Search(c *echo.Context) error {
	term := c.QueryParam("q")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	res, err := r.searchHandler.Handle(c.Request().Context(), queries.SearchVocabularyQuery{
		Term:   term,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo realizar la búsqueda de vocabulario.", err)
	}

	return c.JSON(http.StatusOK, res)
}
