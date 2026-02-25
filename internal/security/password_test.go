package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "SecurePassw0rd!"

	// Тестирование хеширования
	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, password, hashedPassword)

	// Тестирование верификации с правильным паролем
	isValid := VerifyPassword(hashedPassword, password)
	assert.True(t, isValid)

	// Тестирование верификации с неправильным паролем
	isValid = VerifyPassword(hashedPassword, "WrongPassw0rd!")
	assert.False(t, isValid)
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedError string
	}{
		{
			name:          "Valid password",
			password:      "ValidP@ssw0rd",
			expectedError: "",
		},
		{
			name:          "Too short",
			password:      "Short1!",
			expectedError: "минимум 8 символов",
		},
		{
			name:          "No uppercase",
			password:      "noupper1!",
			expectedError: "заглавную букву",
		},
		{
			name:          "No lowercase",
			password:      "NOLOWER1!",
			expectedError: "строчную букву",
		},
		{
			name:          "No digit or special",
			password:      "NoDigitOrSpecial",
			expectedError: "цифру или спецсимвол",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.expectedError), "Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}
