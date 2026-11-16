CREATE TABLE IF NOT EXISTS budget_items (
    id           UUID PRIMARY KEY,
    budget_id    UUID          NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    kind         VARCHAR(20)   NOT NULL,
    reference_id VARCHAR(100)  NOT NULL DEFAULT '',
    description  VARCHAR(255)  NOT NULL,
    quantity     INTEGER       NOT NULL,
    unit_price   DECIMAL(10,2) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budget_items_budget_id ON budget_items (budget_id);
