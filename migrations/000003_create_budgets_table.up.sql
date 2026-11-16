CREATE TABLE IF NOT EXISTS budgets (
    id               UUID PRIMARY KEY,
    order_id         UUID          NOT NULL UNIQUE,
    saga_id          UUID          NOT NULL,
    customer_id      UUID          NOT NULL,
    customer_name    VARCHAR(255)  NOT NULL DEFAULT '',
    customer_email   VARCHAR(255)  NOT NULL DEFAULT '',
    status           VARCHAR(50)   NOT NULL DEFAULT 'pending_approval',
    total_amount     DECIMAL(10,2) NOT NULL DEFAULT 0,
    rejection_reason TEXT          NOT NULL DEFAULT '',
    decided_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budgets_customer_id ON budgets (customer_id);
