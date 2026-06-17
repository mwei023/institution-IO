-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) NOT NULL,
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_roles_institution ON roles(institution_id);

-- Permissions
CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(50) NOT NULL,
    description TEXT,
    UNIQUE (resource, action)
);

-- Role ↔ Permission mapping
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Users
CREATE TABLE IF NOT EXISTS users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    role_id                 UUID NOT NULL REFERENCES roles(id),
    email                   VARCHAR(255),
    phone                   VARCHAR(20),
    first_name              VARCHAR(100) NOT NULL,
    last_name               VARCHAR(100) NOT NULL,
    other_names             VARCHAR(100),
    gender                  CHAR(1) CHECK (gender IN ('M','F','O')),
    date_of_birth           DATE,
    national_id             TEXT,
    profile_photo_url       TEXT,
    password_hash           TEXT NOT NULL,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    mfa_enabled             BOOLEAN NOT NULL DEFAULT false,
    mfa_secret              TEXT,
    login_attempts          SMALLINT NOT NULL DEFAULT 0,
    locked_until            TIMESTAMPTZ,
    last_login_at           TIMESTAMPTZ,
    password_changed_at     TIMESTAMPTZ,
    must_change_password    BOOLEAN NOT NULL DEFAULT true,
    deleted_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID REFERENCES users(id),
    updated_by              UUID REFERENCES users(id),
    CONSTRAINT uq_user_email UNIQUE (institution_id, email),
    CONSTRAINT uq_user_phone UNIQUE (institution_id, phone)
);

CREATE INDEX IF NOT EXISTS idx_users_institution ON users(institution_id, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_email ON users(institution_id, email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_role ON users(institution_id, role_id) WHERE deleted_at IS NULL;

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    token_hash      TEXT NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    user_agent      TEXT,
    ip_address      VARCHAR(45),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(institution_id, user_id);

-- Password history
CREATE TABLE IF NOT EXISTS password_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_history_user ON password_history(institution_id, user_id);
