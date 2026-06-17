-- Chart of accounts
CREATE TABLE IF NOT EXISTS chart_of_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    parent_id       UUID REFERENCES chart_of_accounts(id),
    code            VARCHAR(20) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(20) NOT NULL CHECK (type IN ('asset','liability','equity','income','expense')),
    sub_type        VARCHAR(50),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, code)
);

-- Fee structures
CREATE TABLE IF NOT EXISTS fee_structures (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    class_id            UUID REFERENCES classes(id),
    name                VARCHAR(255) NOT NULL,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fee components
CREATE TABLE IF NOT EXISTS fee_components (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    fee_structure_id    UUID NOT NULL REFERENCES fee_structures(id),
    name                VARCHAR(100) NOT NULL,
    amount              NUMERIC(12,2) NOT NULL,
    term_number         SMALLINT,
    is_mandatory        BOOLEAN NOT NULL DEFAULT true,
    due_date            DATE,
    account_id          UUID REFERENCES chart_of_accounts(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fee invoices
CREATE TABLE IF NOT EXISTS fee_invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    term_id             UUID NOT NULL REFERENCES terms(id),
    invoice_number      VARCHAR(50) NOT NULL,
    total_amount        NUMERIC(12,2) NOT NULL,
    paid_amount         NUMERIC(12,2) NOT NULL DEFAULT 0,
    balance             NUMERIC(12,2) GENERATED ALWAYS AS (total_amount - paid_amount) STORED,
    status              VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                        CHECK (status IN ('unpaid','partial','paid','waived','cancelled')),
    due_date            DATE,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id),
    UNIQUE (institution_id, invoice_number)
);

CREATE INDEX IF NOT EXISTS idx_invoices_student ON fee_invoices(institution_id, student_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invoices_status ON fee_invoices(institution_id, status) WHERE deleted_at IS NULL;

-- Invoice line items
CREATE TABLE IF NOT EXISTS fee_invoice_lines (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    invoice_id      UUID NOT NULL REFERENCES fee_invoices(id),
    component_id    UUID REFERENCES fee_components(id),
    description     VARCHAR(255) NOT NULL,
    amount          NUMERIC(12,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fee payments (receipts)
CREATE TABLE IF NOT EXISTS fee_payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    invoice_id          UUID REFERENCES fee_invoices(id),
    receipt_number      VARCHAR(50) NOT NULL,
    amount              NUMERIC(12,2) NOT NULL,
    payment_method      VARCHAR(30) NOT NULL
                        CHECK (payment_method IN ('mpesa','cash','bank_transfer','cheque','waiver')),
    mpesa_reference     VARCHAR(50),
    bank_reference      VARCHAR(100),
    cheque_number       VARCHAR(50),
    received_by         UUID REFERENCES users(id),
    received_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pdf_url             TEXT,
    is_reversed         BOOLEAN NOT NULL DEFAULT false,
    reversal_reason     TEXT,
    reversed_at         TIMESTAMPTZ,
    reversed_by         UUID REFERENCES users(id),
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, receipt_number)
);

CREATE INDEX IF NOT EXISTS idx_payments_student ON fee_payments(institution_id, student_id) WHERE deleted_at IS NULL;

-- M-Pesa raw transaction log
CREATE TABLE IF NOT EXISTS mpesa_transactions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    checkout_request_id     VARCHAR(100),
    merchant_request_id     VARCHAR(100),
    mpesa_receipt_number    VARCHAR(50),
    amount                  NUMERIC(12,2),
    phone_number            VARCHAR(20),
    account_reference       VARCHAR(100),
    result_code             INTEGER,
    result_description      TEXT,
    raw_payload             JSONB NOT NULL,
    processed               BOOLEAN NOT NULL DEFAULT false,
    payment_id              UUID REFERENCES fee_payments(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, mpesa_receipt_number)
);

-- Journal entries (double-entry)
CREATE TABLE IF NOT EXISTS journal_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    entry_number    VARCHAR(50) NOT NULL,
    entry_date      DATE NOT NULL,
    description     TEXT NOT NULL,
    reference       VARCHAR(100),
    source          VARCHAR(50),
    is_posted       BOOLEAN NOT NULL DEFAULT false,
    posted_at       TIMESTAMPTZ,
    posted_by       UUID REFERENCES users(id),
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id),
    UNIQUE (institution_id, entry_number)
);

CREATE INDEX IF NOT EXISTS idx_journal_entries_date ON journal_entries(institution_id, entry_date DESC) WHERE deleted_at IS NULL;

-- Journal entry lines (debit/credit)
CREATE TABLE IF NOT EXISTS journal_entry_lines (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    entry_id        UUID NOT NULL REFERENCES journal_entries(id),
    account_id      UUID NOT NULL REFERENCES chart_of_accounts(id),
    debit           NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit          NUMERIC(15,2) NOT NULL DEFAULT 0,
    narration       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_debit_credit CHECK (
        (debit > 0 AND credit = 0) OR (credit > 0 AND debit = 0) OR (debit = 0 AND credit = 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_jel_account ON journal_entry_lines(institution_id, account_id);

-- Budgets
CREATE TABLE IF NOT EXISTS budgets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    account_id          UUID NOT NULL REFERENCES chart_of_accounts(id),
    department_id       UUID REFERENCES departments(id),
    budgeted_amount     NUMERIC(15,2) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
    approved_by         UUID REFERENCES users(id),
    approved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);
