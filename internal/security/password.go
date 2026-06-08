package security

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

// VerifyPassword проверяет совпадение хеша и введенного пароля.
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// ValidatePassword проверяет пароль на требования безопасности:
// минимум 8 символов, заглавные и строчные буквы, цифра или спецсимвол
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("пароль должен содержать минимум 8 символов")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case 'A' <= ch && ch <= 'Z':
			hasUpper = true
		case 'a' <= ch && ch <= 'z':
			hasLower = true
		case '0' <= ch && ch <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("пароль должен содержать хотя бы одну заглавную букву")
	}
	if !hasLower {
		return fmt.Errorf("пароль должен содержать хотя бы одну строчную букву")
	}
	if !hasDigit && !hasSpecial {
		return fmt.Errorf("пароль должен содержать хотя бы одну цифру или спецсимвол")
	}

	return nil
}

// HashPassword хеширует пароль с использованием алгоритма bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// GenerateTemporaryPassword создает временный пароль, соответствующий требованиям сложности.
func GenerateTemporaryPassword() (string, error) {
	const letters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const specials = "!@#$%*-_"
	parts := []byte{'A', 'a', '7', '!'}
	alphabet := letters + specials

	for len(parts) < 14 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("failed to generate temporary password: %w", err)
		}
		parts = append(parts, alphabet[n.Int64()])
	}

	for i := len(parts) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", fmt.Errorf("failed to shuffle temporary password: %w", err)
		}
		j := int(n.Int64())
		parts[i], parts[j] = parts[j], parts[i]
	}

	return string(parts), nil
}
