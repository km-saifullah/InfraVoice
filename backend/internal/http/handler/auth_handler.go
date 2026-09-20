package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/auth"
	"github.com/km-saifullah/infra-voice/backend/internal/http/middleware"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
)

type AuthHandler struct {
	service *auth.Service
}

func NewAuthHandler(service *auth.Service) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request registerRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid request body",
		)

		return
	}

	newUser, err := h.service.Register(
		c.Request.Context(),
		auth.RegisterInput{
			Name:     request.Name,
			Email:    request.Email,
			Password: request.Password,
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyExists):
			response.Conflict(
				c,
				"email is already registered",
			)

		case errors.Is(err, auth.ErrInvalidInput):
			response.BadRequest(
				c,
				err.Error(),
			)

		default:
			response.InternalServerError(
				c,
				"failed to create user",
			)
		}

		return
	}

	response.Created(
		c,
		gin.H{
			"user": newUser,
		},
	)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request loginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid request body",
		)

		return
	}

	foundUser, accessToken, err := h.service.Login(
		c.Request.Context(),
		auth.LoginInput{
			Email:    request.Email,
			Password: request.Password,
		},
	)

	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			response.Unauthorized(
				c,
				"invalid email or password",
			)

			return
		}

		response.InternalServerError(
			c,
			"failed to authenticate user",
		)

		return
	}

	response.OK(
		c,
		gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"user":         foundUser,
		},
	)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userIDString, exists := middleware.UserID(c)

	if !exists {
		response.Unauthorized(
			c,
			"authenticated user not found",
		)

		return
	}

	userID, err := bson.ObjectIDFromHex(userIDString)
	if err != nil {
		response.Unauthorized(
			c,
			"invalid authenticated user",
		)

		return
	}

	currentUser, err := h.service.GetUserByID(
		c.Request.Context(),
		userID,
	)

	if err != nil {
		response.InternalServerError(
			c,
			"failed to load authenticated user",
		)

		return
	}

	response.JSON(
		c,
		http.StatusOK,
		gin.H{
			"user": currentUser,
		},
	)
}
