// Package crypto provides authenticated encryption for secrets at rest
// (M-Pesa passkeys, consumer credentials) using AES-256-GCM with the
// application ENCRYPTION_KEY.
//
// Ciphertext format: "enc:v1:<base64(nonce)>:<base64(ciphertext+tag)>"
// The prefix makes encrypted values self-describing: backfill can skip
// already-encrypted rows, and a future key-rotation scheme can add
// enc:v2 without re-reading old formats blind.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// prefix marks values encrypted by this package. Anything not carrying
// the prefix is treated as plaintext legacy data.
const prefix = "enc:v1:"

// ErrNotEncrypted is returned by Decrypt for values without the
// ciphertext prefix — callers that require ciphertext (e.g. key
// rotation) use errors.Is to detect them.
var ErrNotEncrypted = errors.New("value is not encrypted (missing " + prefix + " prefix)")

// ErrBadCiphertext covers malformed or tampered ciphertext: wrong
// prefix layout, bad base64, short payloads, or failed authentication.
var ErrBadCiphertext = errors.New("malformed or tampered ciphertext")

// Encrypt seals plaintext with AES-256-GCM under key (32 raw bytes,
// i.e. a 64-char hex ENCRYPTION_KEY decoded by ParseKey). A fresh
// random nonce is used per call. Empty plaintext round-trips to empty
// string — callers keep NULL/empty semantics instead of storing noise.
func Encrypt(key, plaintext []byte) (string, error) {
	if len(plaintext) == 0 {
		return "", nil
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	return prefix +
		base64.RawStdEncoding.EncodeToString(nonce) + ":" +
		base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a value produced by Encrypt. Values without the
// ciphertext prefix return ErrNotEncrypted (plaintext legacy data —
// callers decide whether to pass through or reject).
func Decrypt(key []byte, value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if !strings.HasPrefix(value, prefix) {
		return nil, ErrNotEncrypted
	}
	rest := strings.TrimPrefix(value, prefix)
	parts := strings.Split(rest, ":")
	if len(parts) != 2 {
		return nil, ErrBadCiphertext
	}
	nonce, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: nonce", ErrBadCiphertext)
	}
	sealed, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload", ErrBadCiphertext)
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed", ErrBadCiphertext)
	}
	return plain, nil
}

// IsEncrypted reports whether a stored value carries this package's
// ciphertext prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefix)
}

// ParseKey decodes a 64-character hex ENCRYPTION_KEY into 32 raw bytes.
func ParseKey(hexKey string) ([]byte, error) {
	key, err := hexDecode(hexKey)
	if err != nil {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 64 hex chars: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

// newGCM builds the AES-256-GCM AEAD for a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// hexDecode is encoding_hex.DecodeString without importing the whole
// package at every call site (kept local for clarity).
func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errors.New("odd-length hex string")
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		hi := hexIndex(s[i])
		lo := hexIndex(s[i+1])
		if hi < 0 || lo < 0 {
			return nil, errors.New("invalid hex byte")
		}
		out[i/2] = byte(hi<<4 | lo)
	}
	return out, nil
}

func hexIndex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
