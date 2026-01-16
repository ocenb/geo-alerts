package utils

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

var ErrInvalidID = errors.New("invalid id format")

func ParseID(c *gin.Context, paramName string) (int64, error) {
	val := c.Param(paramName)
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, ErrInvalidID
	}

	if id <= 0 {
		return 0, ErrInvalidID
	}

	return id, nil
}
