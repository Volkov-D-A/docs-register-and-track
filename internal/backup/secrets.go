package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// SecretBox encrypts settings only. Its key must be backed up separately.
type SecretBox struct {
	aead cipher.AEAD
	id   string
}
type EncryptedSecret struct {
	KeyID string `json:"keyId"`
	Data  string `json:"data"`
}

func LoadSecretBox(path string) (*SecretBox, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read settings key: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("settings key must be base64")
	}
	return NewSecretBox(key)
}
func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("settings key must contain 32 random bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(key)
	return &SecretBox{aead: aead, id: hex.EncodeToString(digest[:8])}, nil
}
func (b *SecretBox) Encrypt(password string) (EncryptedSecret, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return EncryptedSecret{}, err
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(password), []byte("docflow:backup:smb-password:v1:"+b.id))
	return EncryptedSecret{KeyID: b.id, Data: base64.StdEncoding.EncodeToString(sealed)}, nil
}
func (b *SecretBox) Decrypt(secret EncryptedSecret) (string, error) {
	if secret.KeyID != b.id {
		return "", fmt.Errorf("settings key does not match encrypted password")
	}
	raw, err := base64.StdEncoding.DecodeString(secret.Data)
	if err != nil || len(raw) < b.aead.NonceSize()+b.aead.Overhead() {
		return "", fmt.Errorf("invalid encrypted password")
	}
	plaintext, err := b.aead.Open(nil, raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():], []byte("docflow:backup:smb-password:v1:"+b.id))
	if err != nil {
		return "", fmt.Errorf("encrypted password authentication failed")
	}
	return string(plaintext), nil
}
