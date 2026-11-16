CREATE TABLE IF NOT EXISTS payments (
    id                  UUID PRIMARY KEY,
    budget_id           UUID          NOT NULL REFERENCES budgets(id),
    order_id            UUID          NOT NULL UNIQUE,
    saga_id             UUID          NOT NULL,
    amount              DECIMAL(10,2) NOT NULL,
    status              VARCHAR(50)   NOT NULL DEFAULT 'pending',
    provider            VARCHAR(50)   NOT NULL,
    checkout_reference  VARCHAR(255)  NOT NULL DEFAULT '',
    checkout_url        TEXT          NOT NULL DEFAULT '',
    provider_payment_id VARCHAR(255)  NOT NULL DEFAULT '',
    failure_reason      TEXT          NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
