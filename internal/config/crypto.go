package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// encPrefix — префикс, указывающий на зашифрованное значение в конфигурации.
const encPrefix = "ENC:"

// rawEncryptionKey — строка, подставляемая при сборке через ldflags:
//
//	go build -ldflags "-X github.com/Volkov-D-A/docs-register-and-track/internal/config.rawEncryptionKey=<32-байтовый-ключ>"
//
// Не задавайте значение по умолчанию здесь — оно определяется в init().
var rawEncryptionKey string

// encryptionKey — 32-байтовый ключ AES-256 для шифрования/дешифрования конфигурации.
// Инициализируется в init() из rawEncryptionKey (ldflags) или из значения по умолчанию.
var encryptionKey []byte

// defaultDevKey — ключ для режима разработки, когда ldflags не передан.
// В продакшн-сборках всегда используется ключ из ENCRYPTION_KEY (.env → Makefile → ldflags).
const defaultDevKey = "dOcFl0wApp-S3cR3t-K3y!AES256ok!!"

func init() {
	if rawEncryptionKey != "" {
		encryptionKey = []byte(rawEncryptionKey)
	} else {
		encryptionKey = []byte(defaultDevKey)
	}
}

// IsEncrypted проверяет, является ли значение зашифрованным (содержит префикс ENC:).
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encPrefix)
}

// DecryptPassword дешифрует пароль, если он зашифрован (имеет префикс ENC:).
// Если пароль без префикса, возвращает его как есть (обратная совместимость).
func DecryptPassword(value string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(value[len(encPrefix):])
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования base64: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("ошибка создания AES-шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("ошибка создания GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("зашифрованное значение слишком короткое")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка дешифрования: %w", err)
	}

	return string(plaintext), nil
}

// EncryptPassword шифрует пароль и возвращает строку с префиксом ENC: для записи в конфиг.
func EncryptPassword(password string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("ошибка создания AES-шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("ошибка создания GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}
