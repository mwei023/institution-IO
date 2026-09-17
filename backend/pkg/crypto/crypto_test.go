package crypto

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testKey, _ = hex.DecodeString(strings.Repeat("ab", 32))

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	for _, pt := range []string{
		"bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada05671900",
		"short",
		"unicode ✓ secrets",
	} {
		got, err := Encrypt(testKey, []byte(pt))
		require.NoError(t, err)
		assert.True(t, IsEncrypted(got))
		assert.NotContains(t, got, pt, "ciphertext must not leak plaintext")

		plain, err := Decrypt(testKey, got)
		require.NoError(t, err)
		assert.Equal(t, pt, string(plain))
	}
}

func TestEncrypt_UniqueCiphertextPerCall(t *testing.T) {
	a, err := Encrypt(testKey, []byte("same plaintext"))
	require.NoError(t, err)
	b, err := Encrypt(testKey, []byte("same plaintext"))
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "fresh nonce per call — equal ciphertexts would leak equality")
}

func TestEncryptDecrypt_EmptyStaysEmpty(t *testing.T) {
	got, err := Encrypt(testKey, nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	plain, err := Decrypt(testKey, "")
	require.NoError(t, err)
	assert.Empty(t, plain)
	assert.False(t, IsEncrypted(""))
}

func TestDecrypt_PlaintextLegacyValue_ReturnsSentinel(t *testing.T) {
	_, err := Decrypt(testKey, "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada05671900")
	assert.ErrorIs(t, err, ErrNotEncrypted)
}

func TestDecrypt_TamperedCiphertext_Fails(t *testing.T) {
	sealed, err := Encrypt(testKey, []byte("real secret"))
	require.NoError(t, err)

	// Flip one character of the payload.
	tampered := sealed[:len(sealed)-2] + "AA"
	if tampered == sealed {
		t.Fatal("test must actually modify the ciphertext")
	}
	_, err = Decrypt(testKey, tampered)
	assert.ErrorIs(t, err, ErrBadCiphertext)

	// Truncated / malformed forms are rejected, not panicked.
	for _, bad := range []string{prefix, prefix + "!!!:!!", prefix + "AAAA"} {
		_, err := Decrypt(testKey, bad)
		assert.ErrorIs(t, err, ErrBadCiphertext, "value: %q", bad)
	}
}

func TestDecrypt_WrongKey_Fails(t *testing.T) {
	sealed, err := Encrypt(testKey, []byte("secret"))
	require.NoError(t, err)

	otherKey, _ := hex.DecodeString(strings.Repeat("cd", 32))
	_, err = Decrypt(otherKey, sealed)
	assert.ErrorIs(t, err, ErrBadCiphertext)
}

func TestParseKey(t *testing.T) {
	k, err := ParseKey(strings.Repeat("ab", 32))
	require.NoError(t, err)
	assert.Len(t, k, 32)

	_, err = ParseKey("nothex")
	assert.Error(t, err)
	_, err = ParseKey(strings.Repeat("ab", 31)) // 62 chars
	assert.Error(t, err)
}

func TestEncrypt_KeyLengthEnforced(t *testing.T) {
	_, err := Encrypt([]byte("too-short"), []byte("x"))
	assert.Error(t, err)
}
