package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/user"
)

const refreshTokenBytes = 32

const refreshSessionKeyPrefix = "infra-voice:auth:refresh:"

var (
	ErrRefreshTokenInvalid = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)

type SessionStore struct {
	client *goredis.Client
	ttl    time.Duration
}

type refreshSession struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Hash   string `json:"hash"`
}

func NewSessionStore(
	client *goredis.Client,
	cfg config.JWTConfig,
) *SessionStore {
	return &SessionStore{
		client: client,
		ttl:    time.Duration(cfg.RefreshTokenDays) * 24 * time.Hour,
	}
}

func (s *SessionStore) Create(
	ctx context.Context,
	currentUser *user.User,
) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("session store is not initialized")
	}

	if currentUser == nil {
		return "", fmt.Errorf("user is required")
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := hashRefreshToken(refreshToken)

	session := refreshSession{
		UserID: currentUser.ID.Hex(),
		Role:   currentUser.Role,
		Hash:   tokenHash,
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to encode refresh session: %w", err)
	}

	key := refreshSessionKey(tokenHash)

	if err := s.client.Set(
		ctx,
		key,
		payload,
		s.ttl,
	).Err(); err != nil {
		return "", fmt.Errorf("failed to store refresh session: %w", err)
	}

	return refreshToken, nil
}

func (s *SessionStore) Rotate(
	ctx context.Context,
	refreshToken string,
	currentUser *user.User,
) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("session store is not initialized")
	}

	if currentUser == nil {
		return "", fmt.Errorf("user is required")
	}

	refreshToken = strings.TrimSpace(refreshToken)

	if refreshToken == "" {
		return "", ErrRefreshTokenInvalid
	}

	oldHash := hashRefreshToken(refreshToken)
	oldKey := refreshSessionKey(oldHash)

	newToken, err := generateRefreshToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	newHash := hashRefreshToken(newToken)
	newKey := refreshSessionKey(newHash)

	session := refreshSession{
		UserID: currentUser.ID.Hex(),
		Role:   currentUser.Role,
		Hash:   newHash,
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to encode refresh session: %w", err)
	}

	result, err := s.client.Eval(
		ctx,
		rotateRefreshTokenScript,
		[]string{oldKey, newKey},
		string(payload),
		s.ttl.Milliseconds(),
	).Result()

	if err != nil {
		return "", fmt.Errorf("failed to rotate refresh session: %w", err)
	}

	if result == int64(0) {
		return "", ErrRefreshTokenInvalid
	}

	return newToken, nil
}

func (s *SessionStore) GetUserID(
	ctx context.Context,
	refreshToken string,
) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("session store is not initialized")
	}

	refreshToken = strings.TrimSpace(refreshToken)

	if refreshToken == "" {
		return "", ErrRefreshTokenInvalid
	}

	tokenHash := hashRefreshToken(refreshToken)

	payload, err := s.client.Get(
		ctx,
		refreshSessionKey(tokenHash),
	).Bytes()

	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return "", ErrRefreshTokenExpired
		}

		return "", fmt.Errorf("failed to load refresh session: %w", err)
	}

	var session refreshSession

	if err := json.Unmarshal(payload, &session); err != nil {
		return "", fmt.Errorf("failed to decode refresh session: %w", err)
	}

	if session.Hash != tokenHash || session.UserID == "" {
		return "", ErrRefreshTokenInvalid
	}

	return session.UserID, nil
}

func (s *SessionStore) Revoke(
	ctx context.Context,
	refreshToken string,
) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("session store is not initialized")
	}

	refreshToken = strings.TrimSpace(refreshToken)

	if refreshToken == "" {
		return ErrRefreshTokenInvalid
	}

	tokenHash := hashRefreshToken(refreshToken)

	if err := s.client.Del(
		ctx,
		refreshSessionKey(tokenHash),
	).Err(); err != nil {
		return fmt.Errorf("failed to revoke refresh session: %w", err)
	}

	return nil
}

func generateRefreshToken() (string, error) {
	buffer := make([]byte, refreshTokenBytes)

	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func refreshSessionKey(hash string) string {
	return refreshSessionKeyPrefix + hash
}

const rotateRefreshTokenScript = `
local old_key = KEYS[1]
local new_key = KEYS[2]
local payload = ARGV[1]
local ttl_ms = tonumber(ARGV[2])

local old_exists = redis.call("EXISTS", old_key)

if old_exists == 0 then
	return 0
end

redis.call("DEL", old_key)
redis.call("SET", new_key, payload, "PX", ttl_ms)

return 1
`
