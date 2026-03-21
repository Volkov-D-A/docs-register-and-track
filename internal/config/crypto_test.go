package config

import (
	"testing"
)

func init() {
	rawEncryptionKey = "test-key-for-unit-tests-12345678"
}

func TestEncryptDecryptPassword(t *testing.T) {
	// Проверка полного цикла: шифрование пароля, проверка префикса и обратная расшифровка
	original := "MySecretDbPassword123!"

	encrypted, err := EncryptPassword(original)
	if err != nil {
		t.Fatalf("EncryptPassword failed: %v", err)
	}

	if !IsEncrypted(encrypted) {
		t.Fatalf("encrypted value should have ENC: prefix, got: %s", encrypted)
	}

	decrypted, err := DecryptPassword(encrypted)
	if err != nil {
		t.Fatalf("DecryptPassword failed: %v", err)
	}

	if decrypted != original {
		t.Fatalf("decrypted mismatch: expected %q, got %q", original, decrypted)
	}
}

func TestDecryptPassword_PlainText(t *testing.T) {
	// Проверка обработки незашифрованного текста (возвращается без изменений)
	plain := "plainPassword"

	result, err := DecryptPassword(plain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != plain {
		t.Fatalf("plain text should be returned as-is: expected %q, got %q", plain, result)
	}
}

func TestDecryptPassword_InvalidBase64(t *testing.T) {
	// Ошибка: передан невалидный base64, но подходящий под префикс
	_, err := DecryptPassword("ENC:not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptPassword_TooShort(t *testing.T) {
	_, err := DecryptPassword("ENC:AQID") // слишком короткий ciphertext
	if err == nil {
		t.Fatal("expected error for too short ciphertext")
	}
}

func TestEncryptPassword_DifferentEachTime(t *testing.T) {
	// Проверка уникальности шифротекста (соль обеспечивает разный результат для одного и того же пароля)
	password := "SamePassword"

	enc1, err := EncryptPassword(password)
	if err != nil {
		t.Fatalf("first encrypt failed: %v", err)
	}

	enc2, err := EncryptPassword(password)
	if err != nil {
		t.Fatalf("second encrypt failed: %v", err)
	}

	if enc1 == enc2 {
		t.Fatal("two encryptions of the same password should produce different results (random nonce)")
	}

	// Оба должны расшифроваться в одинаковый пароль
	dec1, _ := DecryptPassword(enc1)
	dec2, _ := DecryptPassword(enc2)
	if dec1 != password || dec2 != password {
		t.Fatalf("both should decrypt to %q, got %q and %q", password, dec1, dec2)
	}
}

func TestIsEncrypted(t *testing.T) {
	// Проверка функции-определителя наличия специального префикса ENC:
	if IsEncrypted("plainPassword") {
		t.Fatal("plain text should not be detected as encrypted")
	}
	if !IsEncrypted("ENC:something") {
		t.Fatal("ENC: prefix should be detected as encrypted")
	}
}
