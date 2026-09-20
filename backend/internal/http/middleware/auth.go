package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/auth"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "user_role"
)

func Auth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := strings.TrimSpace(
			c.GetHeader("Authorization"),
		)

		if authorization == "" {
			response.Unauthorized(
				c,
				"authorization token is required",
			)

			c.Abort()

			return
		}

		parts := strings.Fields(authorization)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(
				c,
				"invalid authorization header",
			)

			c.Abort()

			return
		}

		claims, err := jwtService.ParseAccessToken(parts[1])
		if err != nil {
			response.Unauthorized(
				c,
				"invalid or expired access token",
			)

			c.Abort()

			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)

		c.Next()
	}
}

func UserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(ContextUserID)

	if !exists {
		return "", false
	}

	userID, ok := value.(string)

	return userID, ok
}
