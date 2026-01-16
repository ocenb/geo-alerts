package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/http/response"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

// HealthCheck godoc
// @Summary      Health check
// @Description  Checks if the API service is running.
// @Tags         system
// @Produce      json
// @Success      200  {object}  models.HealthCheckResult
// @Router       /system/health [get]
func (h *Handler) health(c *gin.Context) {
	response.OK(c, models.HealthCheckResult{Status: "OK"})
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	systemRouter := router.Group("/system")
	systemRouter.GET("/health", h.health)
}
