-- migrate:up
SET search_path TO transactions, public;
CREATE TABLE category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_category_code ON category(code);
CREATE INDEX idx_category_is_active ON category(is_active);
ALTER TABLE transaction ADD COLUMN category_id BIGINT REFERENCES category(id);
CREATE INDEX idx_transaction_category_id ON transaction(category_id);
DROP INDEX IF EXISTS idx_transaction_budget_category_id;
-- Intentional destructive replacement: budget_category mixed category assignment
-- with budget allocation, and no mapping is preserved to the new global category table.
ALTER TABLE transaction DROP COLUMN budget_category_id;
DROP TABLE budget_category;

-- migrate:down
SET search_path TO transactions, public;
CREATE TABLE budget_category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_id BIGINT,
    category_name VARCHAR NOT NULL,
    allocation REAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (budget_id) REFERENCES budget(id)
);
ALTER TABLE transaction ADD COLUMN budget_category_id BIGINT REFERENCES budget_category(id);
CREATE INDEX idx_transaction_budget_category_id ON transaction(budget_category_id);
DROP INDEX IF EXISTS idx_transaction_category_id;
ALTER TABLE transaction DROP COLUMN category_id;
DROP INDEX IF EXISTS idx_category_is_active;
DROP INDEX IF EXISTS idx_category_code;
DROP TABLE category;
