package tests

// Integration test for M-Pesa channel secret encryption at rest — the
// acceptance check for the pkg/crypto + migration 000017 + backfill
// pipeline. Unit tests pin the crypto primitives and the decrypt gate;
// this test proves the whole chain against real Postgres:
//
//  1. storing "enc:v1:..." ciphertext (as cmd/backfill-channels writes it)
//     satisfies the table's CHECK constraints;
//  2. GetDefaultChannel decrypts on read and returns the plaintext secret;
//  3. without the key wired, the same row is refused (never silently served).
//
// Skips when DATABASE_URL is unset (same convention as the other
// integration tests). Fixture is isolated and cascades away in cleanup.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"institution-os/internal/finance"
	"institution-os/pkg/crypto"
	"institution-os/pkg/db"
)

func TestChannelSecrets_EncryptedAtRestRoundTrip(t *testing.T) {
	pool := intDB(t)
	f := newFinanceFixture(t)

	key, err := crypto.ParseKey("3333333333333333333333333333333333333333333333333333333333333333")
	require.NoError(t, err)

	cipher, err := crypto.Encrypt(key, []byte("daraja-passkey-roundtrip"))
	require.NoError(t, err)
	require.True(t, crypto.IsEncrypted(cipher))

	// Insert ciphertext directly — exactly the post-backfill storage format.
	_, err = pool.Exec(f.ctx, `
		INSERT INTO payment_channels (id, institution_id, channel_type, name, shortcode, passkey, is_default, is_active)
		VALUES ($1, $2, 'mpesa_till', 'Enc Roundtrip Till', $3, $4, true, true)`,
		uuid.New(), f.instID, "RT-"+f.instID.String()[:13], cipher)
	require.NoError(t, err) // CHECK constraints accept enc:v1: ciphertext

	// With the key: decrypt-on-read returns the original secret.
	repo := finance.NewRepository(&db.Pool{Pool: pool})
	repo.SetEncryptionKey(key)
	ch, err := repo.GetDefaultChannel(f.ctx, f.instID)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, "daraja-passkey-roundtrip", *ch.Passkey)

	// Without the key: the encrypted row must be refused, not served garbled.
	repoNoKey := finance.NewRepository(&db.Pool{Pool: pool})
	_, err = repoNoKey.GetDefaultChannel(f.ctx, f.instID)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no ENCRYPTION_KEY is wired")
}
