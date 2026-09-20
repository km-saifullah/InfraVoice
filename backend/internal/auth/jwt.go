package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/user"
)

type JWTService struct {
	secret             []byte
	accessTokenMinutes int
}

type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		secret:             []byte(cfg.Secret),
		accessTokenMinutes: cfg.AccessTokenMinutes,
	}
}

func (s *JWTService) GenerateAccessToken(
	currentUser *user.User,
) (string, error) {
	if currentUser == nil {
		return "", fmt.Errorf("user is required")
	}

	now := time.Now()

	claims := Claims{
		UserID: currentUser.ID.Hex(),
		Role:   currentUser.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  currentUser.ID.Hex(),
			Issuer:   "infra-voice-api",
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(
					time.Duration(s.accessTokenMinutes) * time.Minute,
				),
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

func (s *JWTService) ParseAccessToken(
	tokenString string,
) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is required")
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %s",
					token.Method.Alg(),
				)
			}

			return s.secret, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid access token claims")
	}

	if _, err := bson.ObjectIDFromHex(claims.UserID); err != nil {
		return nil, fmt.Errorf("invalid user id in token")
	}

	return claims, nil
}
