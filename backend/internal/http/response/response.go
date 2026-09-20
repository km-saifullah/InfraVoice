package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func JSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	JSON(c, http.StatusCreated, data)
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, data)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(
	c *gin.Context,
	statusCode int,
	code string,
	message string,
) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func ErrorWithDetails(
	c *gin.Context,
	statusCode int,
	code string,
	message string,
	details interface{},
) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
