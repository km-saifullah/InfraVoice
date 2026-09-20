package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	if len([]byte(password)) > 72 {
		return "", fmt.Errorf("password exceeds bcrypt maximum length")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcryptCost,
	)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

func ComparePassword(
	passwordHash string,
	password string,
) error {
	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	); err != nil {
		return fmt.Errorf("invalid password")
	}

	return nil
}
