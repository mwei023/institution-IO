CREATE TABLE IF NOT EXISTS institutions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(255) NOT NULL,
    short_name              VARCHAR(50),
    type                    VARCHAR(50) NOT NULL DEFAULT 'secondary',
    registration_number     VARCHAR(100),
    logo_url                TEXT,
    address                 TEXT,
    county                  VARCHAR(100),
    sub_county              VARCHAR(100),
    phone                   VARCHAR(20),
    email                   VARCHAR(255),
    website_url             TEXT,
    school_slug             VARCHAR(100) UNIQUE,
    settings                JSONB NOT NULL DEFAULT '{}',
    subscription_plan       VARCHAR(50) NOT NULL DEFAULT 'starter',
    subscription_expires_at TIMESTAMPTZ,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_institutions_slug ON institutions(school_slug) WHERE is_active = true;
