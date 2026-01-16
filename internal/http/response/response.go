package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func OK(c *gin.Context, obj any) {
	c.JSON(http.StatusOK, obj)
}

func Created(c *gin.Context, obj any) {
	c.JSON(http.StatusCreated, obj)
}

func InternalError(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Message: "Internal server error"})
}

func NotFoundError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Message: msg})
}

func BadRequestError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Message: msg})
}

func ConflictError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusConflict, ErrorResponse{Message: msg})
}

func ForbiddenError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Message: "Forbidden"})
}
