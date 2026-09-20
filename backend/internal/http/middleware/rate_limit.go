package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
)

const rateLimitWindow = time.Minute

func RateLimit(
	client *goredis.Client,
	maxRequests int,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil || maxRequests <= 0 {
			c.Next()

			return
		}

		clientIP := c.ClientIP()

		window := time.Now().Unix() / int64(rateLimitWindow.Seconds())

		key := fmt.Sprintf(
			"infra-voice:rate-limit:%s:%d",
			clientIP,
			window,
		)

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			500*time.Millisecond,
		)
		defer cancel()

		count, err := client.Incr(ctx, key).Result()

		if err != nil {
			c.Next()

			return
		}

		if count == 1 {
			_ = client.Expire(ctx, key, rateLimitWindow+time.Second).Err()
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, maxRequests-int(count))))

		if count > int64(maxRequests) {
			c.Header("Retry-After", "60")

			response.Error(
				c,
				http.StatusTooManyRequests,
				response.CodeTooManyRequests,
				"rate limit exceeded",
			)

			c.Abort()

			return
		}

		c.Next()
	}
}

func max(first, second int) int {
	if first > second {
		return first
	}

	return second
}
