package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Serahkan ke handler berikutnya dulu.
		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		attrs := []any{
			"request_id", GetRequestID(c),
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case status >= 500:
			log.Error("request selesai", attrs...)
		case status >= 400:
			log.Warn("request selesai", attrs...)
		default:
			log.Info("request selesai", attrs...)
		}
	}
}

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic tertangkap",
					"request_id", GetRequestID(c),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"error", err,
				)
				c.AbortWithStatusJSON(500, gin.H{
					"error":      "terjadi kesalahan internal",
					"request_id": GetRequestID(c),
				})
			}
		}()
		c.Next()
	}
}