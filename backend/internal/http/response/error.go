package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeTooManyRequests    = "TOO_MANY_REQUESTS"
	CodePayloadTooLarge    = "PAYLOAD_TOO_LARGE"
	CodeInternalServer     = "INTERNAL_SERVER_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

func BadRequest(c *gin.Context, message string) {
	Error(
		c,
		http.StatusBadRequest,
		CodeBadRequest,
		message,
	)
}

func Unauthorized(c *gin.Context, message string) {
	Error(
		c,
		http.StatusUnauthorized,
		CodeUnauthorized,
		message,
	)
}

func Forbidden(c *gin.Context, message string) {
	Error(
		c,
		http.StatusForbidden,
		CodeForbidden,
		message,
	)
}

func NotFound(c *gin.Context, message string) {
	Error(
		c,
		http.StatusNotFound,
		CodeNotFound,
		message,
	)
}

func Conflict(c *gin.Context, message string) {
	Error(
		c,
		http.StatusConflict,
		CodeConflict,
		message,
	)
}

func TooManyRequests(c *gin.Context, message string) {
	Error(
		c,
		http.StatusTooManyRequests,
		CodeTooManyRequests,
		message,
	)
}

func PayloadTooLarge(c *gin.Context, message string) {
	Error(
		c,
		http.StatusRequestEntityTooLarge,
		CodePayloadTooLarge,
		message,
	)
}

func InternalServerError(c *gin.Context, message string) {
	Error(
		c,
		http.StatusInternalServerError,
		CodeInternalServer,
		message,
	)
}

func ServiceUnavailable(c *gin.Context, message string) {
	Error(
		c,
		http.StatusServiceUnavailable,
		CodeServiceUnavailable,
		message,
	)
}
