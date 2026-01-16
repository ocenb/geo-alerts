package location

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/http/response"
)

type Service interface {
	Check(ctx context.Context, params *models.CheckLocationParams) (*models.CheckLocationResult, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CheckLocation godoc
// @Summary      Check user location
// @Description  Checks if the user's coordinates are within any active dangerous zone.
// @Tags         location
// @Accept       json
// @Produce      json
// @Param        input body CheckReq true "Location parameters"
// @Success      201  {object}  models.CheckLocationResult
// @Failure      400  {object}  response.ErrorResponse "Invalid input"
// @Failure      500  {object}  response.ErrorResponse "Internal server error"
// @Router       /location/check [post]
func (h *Handler) check(c *gin.Context) {
	var req CheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	params := &models.CheckLocationParams{
		UserID:    req.UserID,
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
	}

	res, err := h.service.Check(c.Request.Context(), params)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Created(c, res)
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	locationRouter := router.Group("/location")
	locationRouter.POST("/check", h.check)
}
