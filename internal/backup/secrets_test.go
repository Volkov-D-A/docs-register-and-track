package backup

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSecretBoxRoundtripTamperingAndWrongKey(t *testing.T) {
	box, err := NewSecretBox(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	secret, err := box.Encrypt("пароль ' $HOME\n")
	require.NoError(t, err)
	plain, err := box.Decrypt(secret)
	require.NoError(t, err)
	require.Equal(t, "пароль ' $HOME\n", plain)
	second, err := box.Encrypt(plain)
	require.NoError(t, err)
	require.NotEqual(t, secret.Data, second.Data)
	other, err := NewSecretBox(bytes.Repeat([]byte{2}, 32))
	require.NoError(t, err)
	_, err = other.Decrypt(secret)
	require.Error(t, err)
	secret.Data = "AAAA" + secret.Data[4:]
	_, err = box.Decrypt(secret)
	require.Error(t, err)
	_, err = NewSecretBox([]byte("short"))
	require.Error(t, err)
}
