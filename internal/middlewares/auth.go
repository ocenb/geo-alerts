package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Auth(requiredKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientKey := c.GetHeader("X-API-Key")

		if clientKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API key is missing",
			})
			return
		}

		if clientKey != requiredKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			return
		}

		c.Next()
	}
}
