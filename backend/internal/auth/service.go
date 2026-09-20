package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/km-saifullah/infra-voice/backend/internal/user"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
)

type Service struct {
	users    *user.Repository
	jwt      *JWTService
	sessions *SessionStore
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type RefreshInput struct {
	RefreshToken string
}

func NewService(
	users *user.Repository,
	jwtService *JWTService,
	sessionStore *SessionStore,
) *Service {
	return &Service{
		users:    users,
		jwt:      jwtService,
		sessions: sessionStore,
	}
}

func (s *Service) Register(
	ctx context.Context,
	input RegisterInput,
) (*user.User, error) {
	name := strings.TrimSpace(input.Name)
	email := normalizeEmail(input.Email)

	if name == "" {
		return nil, fmt.Errorf(
			"%w: name is required",
			ErrInvalidInput,
		)
	}

	if len(name) > 100 {
		return nil, fmt.Errorf(
			"%w: name must not exceed 100 characters",
			ErrInvalidInput,
		)
	}

	if err := validateEmail(email); err != nil {
		return nil, fmt.Errorf(
			"%w: invalid email",
			ErrInvalidInput,
		)
	}

	if err := validatePassword(input.Password); err != nil {
		return nil, err
	}

	existingUser, err := s.users.FindByEmail(ctx, email)

	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	newUser := &user.User{
		ID:           bson.NewObjectID(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         user.RoleUser,
		Status:       user.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrEmailAlreadyExists
		}

		return nil, err
	}

	return newUser, nil
}

func (s *Service) Login(
	ctx context.Context,
	input LoginInput,
) (*user.User, string, string, error) {
	email := normalizeEmail(input.Email)

	if err := validateEmail(email); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	if input.Password == "" {
		return nil, "", "", ErrInvalidCredentials
	}

	foundUser, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, "", "", ErrInvalidCredentials
		}

		return nil, "", "", err
	}

	if !foundUser.IsActive() {
		return nil, "", "", ErrInvalidCredentials
	}

	if err := ComparePassword(
		foundUser.PasswordHash,
		input.Password,
	); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	accessToken, err := s.jwt.GenerateAccessToken(foundUser)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := s.sessions.Create(
		ctx,
		foundUser,
	)
	if err != nil {
		return nil, "", "", err
	}

	return foundUser, accessToken, refreshToken, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	input RefreshInput,
) (*user.User, string, string, error) {
	userIDString, err := s.sessions.GetUserID(
		ctx,
		input.RefreshToken,
	)
	if err != nil {
		return nil, "", "", err
	}

	userID, err := bson.ObjectIDFromHex(userIDString)
	if err != nil {
		return nil, "", "", ErrRefreshTokenInvalid
	}

	currentUser, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, "", "", ErrRefreshTokenInvalid
		}

		return nil, "", "", err
	}

	if !currentUser.IsActive() {
		return nil, "", "", ErrRefreshTokenInvalid
	}

	accessToken, err := s.jwt.GenerateAccessToken(currentUser)
	if err != nil {
		return nil, "", "", err
	}

	newRefreshToken, err := s.sessions.Rotate(
		ctx,
		input.RefreshToken,
		currentUser,
	)
	if err != nil {
		return nil, "", "", err
	}

	return currentUser, accessToken, newRefreshToken, nil
}

func (s *Service) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	return s.sessions.Revoke(
		ctx,
		refreshToken,
	)
}

func (s *Service) GetUserByID(
	ctx context.Context,
	id bson.ObjectID,
) (*user.User, error) {
	return s.users.FindByID(ctx, id)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) error {
	address, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	if !strings.EqualFold(address.Address, email) {
		return fmt.Errorf("invalid email")
	}

	return nil
}

func validatePassword(password string) error {
	passwordLength := len([]byte(password))

	if passwordLength < 8 {
		return fmt.Errorf(
			"%w: password must contain at least 8 bytes",
			ErrInvalidInput,
		)
	}

	if passwordLength > 72 {
		return fmt.Errorf(
			"%w: password must not exceed 72 bytes",
			ErrInvalidInput,
		)
	}

	return nil
}
