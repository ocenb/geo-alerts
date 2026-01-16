package middlewares

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logging(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		reqID := c.Writer.Header().Get("X-Request-ID")

		reqLog := log.With(
			slog.String("req_id", reqID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", raw),
			slog.String("user_agent", c.Request.UserAgent()),
		)

		reqLog.Debug("Request")

		c.Next()

		status := c.Writer.Status()

		reqLog.Info("Response",
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
		)
	}
}
