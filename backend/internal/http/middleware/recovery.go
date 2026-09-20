package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get("request_id")

				logger.Error(
					"panic recovered",
					"request_id", requestID,
					"panic", recovered,
				)

				response.Error(
					c,
					http.StatusInternalServerError,
					response.CodeInternalServer,
					"an unexpected error occurred",
				)

				c.Abort()
			}
		}()

		c.Next()
	}
}
