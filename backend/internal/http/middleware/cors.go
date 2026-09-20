package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	origins := make(map[string]struct{}, len(allowedOrigins))

	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)

		if origin != "" {
			origins[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if _, allowed := origins[origin]; allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header(
				"Access-Control-Allow-Headers",
				"Origin, Content-Type, Accept, Authorization, X-Request-ID",
			)
			c.Header(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)
		}

		if c.Request.Method == http.MethodOptions {
			if _, allowed := origins[origin]; allowed {
				c.Status(http.StatusNoContent)
				c.Abort()

				return
			}

			c.Status(http.StatusForbidden)
			c.Abort()

			return
		}

		c.Next()
	}
}
