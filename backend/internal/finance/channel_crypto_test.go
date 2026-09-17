package finance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"institution-os/pkg/crypto"
)

// decryptChannelSecret is the read-side gate for channel secrets at rest.
// These tests pin its paths: empty passthrough, legacy plaintext
// passthrough with warning, encrypted value opened under the wired key,
// and encrypted value rejected (not served silently) without a key.
func TestDecryptChannelSecret(t *testing.T) {
	key, err := crypto.ParseKey("1111111111111111111111111111111111111111111111111111111111111111")
	require.NoError(t, err)

	enc, err := crypto.Encrypt(key, []byte("super-secret-passkey"))
	require.NoError(t, err)
	require.True(t, crypto.IsEncrypted(enc))

	t.Run("empty value passes through as empty", func(t *testing.T) {
		r := &Repository{}
		v, err := r.decryptChannelSecret("passkey", "")
		require.NoError(t, err)
		assert.Empty(t, v)
	})

	t.Run("legacy plaintext passes through with warning", func(t *testing.T) {
		r := &Repository{}
		v, err := r.decryptChannelSecret("passkey", "raw-legacy-passkey")
		require.NoError(t, err)
		assert.Equal(t, "raw-legacy-passkey", v)
	})

	t.Run("encrypted value opens under wired key", func(t *testing.T) {
		r := &Repository{}
		r.SetEncryptionKey(key)
		v, err := r.decryptChannelSecret("passkey", enc)
		require.NoError(t, err)
		assert.Equal(t, "super-secret-passkey", v)
	})

	t.Run("encrypted value without key fails loudly", func(t *testing.T) {
		r := &Repository{}
		_, err := r.decryptChannelSecret("passkey", enc)
		require.Error(t, err)
		assert.ErrorContains(t, err, "no ENCRYPTION_KEY is wired")
	})

	t.Run("wrong key fails with tampered error", func(t *testing.T) {
		other, err := crypto.ParseKey("2222222222222222222222222222222222222222222222222222222222222222")
		require.NoError(t, err)
		r := &Repository{}
		r.SetEncryptionKey(other)
		_, err = r.decryptChannelSecret("passkey", enc)
		require.Error(t, err)
		assert.ErrorIs(t, err, crypto.ErrBadCiphertext)
	})

	t.Run("ciphertext tampering fails authentication", func(t *testing.T) {
		r := &Repository{}
		r.SetEncryptionKey(key)
		tampered := enc[:len(enc)-4] + "AAAA"
		_, err := r.decryptChannelSecret("consumer_secret", tampered)
		require.Error(t, err)
		assert.ErrorIs(t, err, crypto.ErrBadCiphertext)
	})
}
