-- payment_channels: till/paybill/bank accounts an institution receives
-- money through. This table existed only on manually-migrated databases
-- (created outside the repo, like the pre-000008 drift); this migration
-- recovers it so fresh staging/pilot databases are not missing it.
--
-- Sensitive columns (passkey, consumer_key, consumer_secret) are stored
-- ENCRYPTED AT REST with AES-256-GCM under ENCRYPTION_KEY, prefixed
-- "enc:v1:" — see pkg/crypto and cmd/backfill-channels. Shortcode stays
-- plaintext: it is a public identifier and a lookup key
-- (GetChannelByShortcode). The CHECK keeps already-encrypted values
-- valid while preventing new plaintext secrets from being inserted.
CREATE TABLE IF NOT EXISTS payment_channels (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    channel_type        VARCHAR(20) NOT NULL
                        CHECK (channel_type IN ('mpesa_paybill','mpesa_till','bank_account')),
    name                VARCHAR(100) NOT NULL,
    shortcode           VARCHAR(20),
    passkey             TEXT CHECK (passkey IS NULL OR passkey LIKE 'enc:v1:%'),
    consumer_key        TEXT CHECK (consumer_key IS NULL OR consumer_key LIKE 'enc:v1:%'),
    consumer_secret     TEXT CHECK (consumer_secret IS NULL OR consumer_secret LIKE 'enc:v1:%'),
    bank_name           VARCHAR(100),
    bank_account_number VARCHAR(50),
    bank_account_name   VARCHAR(255),
    is_default          BOOLEAN NOT NULL DEFAULT false,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_channels_institution
    ON payment_channels(institution_id) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_channels_default
    ON payment_channels(institution_id)
    WHERE is_default = true AND is_active = true AND deleted_at IS NULL;

-- On databases where the table pre-existed (created outside this repo —
-- the same drift that produced the 000008 gap), CREATE TABLE IF NOT
-- EXISTS above skipped the inline CHECKs. Add them if missing, NOT
-- VALID: existing plaintext rows stay readable until the backfill
-- (cmd/backfill-channels) rewrites them, but any NEW write of a
-- plaintext secret is rejected. Run VALIDATE after backfilling.
DO $$
DECLARE
    c text;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'payment_channels') THEN
        -- Separator is '|' — ':' cannot be used because 'enc:v1:' contains it.
        FOREACH c IN ARRAY ARRAY[
            'chk_channels_passkey_encrypted|CHECK (passkey IS NULL OR passkey LIKE ''enc:v1:%'')',
            'chk_channels_consumer_key_encrypted|CHECK (consumer_key IS NULL OR consumer_key LIKE ''enc:v1:%'')',
            'chk_channels_consumer_secret_encrypted|CHECK (consumer_secret IS NULL OR consumer_secret LIKE ''enc:v1:%'')'
        ] LOOP
            IF NOT EXISTS (
                SELECT 1 FROM pg_constraint
                WHERE conname = split_part(c, '|', 1)
                  AND conrelid = 'payment_channels'::regclass
            ) THEN
                EXECUTE 'ALTER TABLE payment_channels ADD CONSTRAINT '
                    || split_part(c, '|', 1) || ' ' || split_part(c, '|', 2) || ' NOT VALID';
            END IF;
        END LOOP;
    END IF;
END $$;
