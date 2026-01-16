package incident

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/http/response"
	"github.com/ocenb/geo-alerts/internal/http/utils"
)

type Service interface {
	Create(ctx context.Context, params *models.CreateIncidentParams) (*models.Incident, error)
	GetByID(ctx context.Context, id int64) (*models.Incident, error)
	List(ctx context.Context, limit, offset int) ([]models.Incident, error)
	Update(ctx context.Context, params *models.UpdateIncidentParams) (*models.Incident, error)
	Deactivate(ctx context.Context, id int64) error
	GetStats(ctx context.Context) ([]models.Stats, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateIncident godoc
// @Summary      Create a new incident
// @Description  Creates a dangerous zone incident. Returns the created incident.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        input body CreateReq true "Incident parameters"
// @Success      201  {object}  models.Incident
// @Failure      400  {object}  response.ErrorResponse "Invalid input"
// @Failure      409  {object}  response.ErrorResponse "Incident already exists in this area"
// @Failure      500  {object}  response.ErrorResponse "Internal server error"
// @Router       /incidents [post]
func (h *Handler) create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	params := &models.CreateIncidentParams{
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
		Radius:    req.Radius,
	}

	inc, err := h.service.Create(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, errs.ErrIncidentExists) {
			response.ConflictError(c, "Incident already exists in this area")
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, inc)
}

// GetIncident godoc
// @Summary      Get incident by ID
// @Description  Returns detailed information about a specific incident.
// @Tags         incidents
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      int  true  "Incident ID"
// @Success      200  {object}  models.Incident
// @Failure      400  {object}  response.ErrorResponse "Invalid ID format"
// @Failure      404  {object}  response.ErrorResponse "Incident not found"
// @Failure      500  {object}  response.ErrorResponse "Internal server error"
// @Router       /incidents/{id} [get]
func (h *Handler) getByID(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	inc, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrIncidentNotFound) {
			response.NotFoundError(c, "Incident not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, inc)
}

// ListIncidents godoc
// @Summary      List incidents
// @Description  Get a paginated list of incidents.
// @Tags         incidents
// @Produce      json
// @Security     ApiKeyAuth
// @Param        limit   query     int  false  "Limit (default 10)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200     {array}   models.Incident
// @Failure      400     {object}  response.ErrorResponse "Invalid query parameters"
// @Failure      500     {object}  response.ErrorResponse "Internal server error"
// @Router       /incidents [get]
func (h *Handler) list(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	incidents, err := h.service.List(c.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, incidents)
}

// UpdateIncident godoc
// @Summary      Update incident
// @Description  Updates location or radius of an existing incident.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id     path      int        true  "Incident ID"
// @Param        input  body      UpdateReq  true  "Update parameters"
// @Success      200    {object}  models.Incident
// @Failure      400    {object}  response.ErrorResponse "Invalid input or ID"
// @Failure      404    {object}  response.ErrorResponse "Incident not found"
// @Failure      409    {object}  response.ErrorResponse "Conflict with another incident"
// @Failure      500    {object}  response.ErrorResponse "Internal server error"
// @Router       /incidents/{id} [put]
func (h *Handler) update(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	params := &models.UpdateIncidentParams{
		ID:        id,
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
		Radius:    req.Radius,
	}

	inc, err := h.service.Update(c.Request.Context(), params)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrIncidentNotFound):
			response.NotFoundError(c, "Incident not found")
		case errors.Is(err, errs.ErrIncidentExists):
			response.ConflictError(c, "Incident conflict: location overlaps with another incident")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, inc)
}

// DeleteIncident godoc
// @Summary      Deactivate incident
// @Description  Soft deletes (deactivates) an incident by ID.
// @Tags         incidents
// @Security     ApiKeyAuth
// @Param        id   path      int  true  "Incident ID"
// @Success      204  "No Content"
// @Failure      400  {object}  response.ErrorResponse "Invalid ID format"
// @Failure      404  {object}  response.ErrorResponse "Incident not found"
// @Failure      500  {object}  response.ErrorResponse "Internal server error"
// @Router       /incidents/{id} [delete]
func (h *Handler) delete(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}

	err = h.service.Deactivate(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrIncidentNotFound) {
			response.NotFoundError(c, "Incident not found")
			return
		}
		response.InternalError(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetStats godoc
// @Summary      Get incident statistics
// @Description  Returns statistics regarding unique users near dangerous zones.
// @Tags         incidents
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   models.Stats
// @Failure      500  {object}  response.ErrorResponse "Internal server error"
// @Router       /incidents/stats [get]
func (h *Handler) getStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, stats)
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	incRouter := router.Group("/incidents")
	incRouter.POST("", h.create)
	incRouter.GET(":id", h.getByID)
	incRouter.GET("", h.list)
	incRouter.PUT(":id", h.update)
	incRouter.DELETE(":id", h.delete)
	incRouter.GET("/stats", h.getStats)
}
