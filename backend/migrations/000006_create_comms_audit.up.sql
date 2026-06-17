-- Announcements
CREATE TABLE IF NOT EXISTS announcements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    type            VARCHAR(30) NOT NULL DEFAULT 'general',
    audience        VARCHAR(50) NOT NULL DEFAULT 'all',
    send_sms        BOOLEAN NOT NULL DEFAULT false,
    send_push       BOOLEAN NOT NULL DEFAULT true,
    is_published    BOOLEAN NOT NULL DEFAULT false,
    published_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    attachments     JSONB,
    read_count      INTEGER NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_announcements_published ON announcements(institution_id, published_at DESC)
    WHERE deleted_at IS NULL AND is_published = true;

-- Announcement read tracking
CREATE TABLE IF NOT EXISTS announcement_reads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    announcement_id UUID NOT NULL REFERENCES announcements(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    read_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, announcement_id, user_id)
);

-- Direct messages (parent ↔ teacher, staff ↔ staff)
CREATE TABLE IF NOT EXISTS messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    sender_id       UUID NOT NULL REFERENCES users(id),
    recipient_id    UUID NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,
    is_read         BOOLEAN NOT NULL DEFAULT false,
    read_at         TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(institution_id, recipient_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Notifications log
CREATE TABLE IF NOT EXISTS notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    type            VARCHAR(50) NOT NULL,
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    channel         VARCHAR(20) NOT NULL,
    is_read         BOOLEAN NOT NULL DEFAULT false,
    read_at         TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    failed_at       TIMESTAMPTZ,
    failure_reason  TEXT,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(institution_id, user_id, created_at DESC);

-- Digital locker folders
CREATE TABLE IF NOT EXISTS locker_folders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    entity_type     VARCHAR(30) NOT NULL,
    entity_id       UUID NOT NULL,
    name            VARCHAR(100) NOT NULL,
    parent_folder_id UUID REFERENCES locker_folders(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_locker_folders_entity ON locker_folders(institution_id, entity_type, entity_id);

-- Digital locker documents
CREATE TABLE IF NOT EXISTS locker_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    folder_id       UUID NOT NULL REFERENCES locker_folders(id),
    entity_type     VARCHAR(30) NOT NULL,
    entity_id       UUID NOT NULL,
    document_type   VARCHAR(50),
    title           VARCHAR(255) NOT NULL,
    file_name       VARCHAR(255) NOT NULL,
    file_size       INTEGER,
    mime_type       VARCHAR(100),
    storage_key     TEXT NOT NULL,
    version         SMALLINT NOT NULL DEFAULT 1,
    previous_version_id UUID REFERENCES locker_documents(id),
    access_level    VARCHAR(20) NOT NULL DEFAULT 'institution',
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_locker_docs_entity ON locker_documents(institution_id, entity_type, entity_id)
    WHERE deleted_at IS NULL;

-- IMMUTABLE audit log (append-only enforced via rules)
CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID,
    user_id         UUID,
    action          VARCHAR(100) NOT NULL,
    table_name      VARCHAR(100),
    record_id       UUID,
    old_values      JSONB,
    new_values      JSONB,
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    request_id      VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_institution ON audit_logs(institution_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_record ON audit_logs(table_name, record_id, created_at DESC);

-- Prevent anyone from updating or deleting audit records
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_rules WHERE tablename = 'audit_logs' AND rulename = 'no_update_audit'
    ) THEN
        CREATE RULE no_update_audit AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_rules WHERE tablename = 'audit_logs' AND rulename = 'no_delete_audit'
    ) THEN
        CREATE RULE no_delete_audit AS ON DELETE TO audit_logs DO INSTEAD NOTHING;
    END IF;
END$$;

-- Workflow definitions
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    entity_type     VARCHAR(50) NOT NULL,
    steps           JSONB NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Workflow instances
CREATE TABLE IF NOT EXISTS workflow_instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    definition_id   UUID NOT NULL REFERENCES workflow_definitions(id),
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       UUID NOT NULL,
    current_step    SMALLINT NOT NULL DEFAULT 1,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    initiated_by    UUID NOT NULL REFERENCES users(id),
    completed_at    TIMESTAMPTZ,
    history         JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_instances_entity ON workflow_instances(institution_id, entity_type, entity_id);
