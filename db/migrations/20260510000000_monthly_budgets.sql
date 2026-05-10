-- migrate:up
SET search_path TO transactions, public;

ALTER TABLE budget DROP COLUMN type;
ALTER TABLE budget ADD COLUMN period_start DATE;
ALTER TABLE budget ADD COLUMN period_end DATE;
ALTER TABLE budget ADD COLUMN source_budget_id BIGINT REFERENCES budget(id);

UPDATE budget
SET
    period_start = CURRENT_DATE,
    period_end = CURRENT_DATE
WHERE period_start IS NULL OR period_end IS NULL;

ALTER TABLE budget ALTER COLUMN period_start SET NOT NULL;
ALTER TABLE budget ALTER COLUMN period_end SET NOT NULL;

ALTER TABLE budget ADD CONSTRAINT chk_budget_exactly_one_owner
CHECK (
    (household_id IS NOT NULL AND user_id IS NULL)
    OR
    (household_id IS NULL AND user_id IS NOT NULL)
);

ALTER TABLE budget ADD CONSTRAINT chk_budget_valid_period
CHECK (period_end >= period_start);

CREATE UNIQUE INDEX idx_budget_household_period
ON budget(household_id, period_start, period_end)
WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_budget_user_period
ON budget(user_id, period_start, period_end)
WHERE user_id IS NOT NULL;

CREATE INDEX idx_budget_household_period_start
ON budget(household_id, period_start)
WHERE household_id IS NOT NULL;

CREATE INDEX idx_budget_user_period_start
ON budget(user_id, period_start)
WHERE user_id IS NOT NULL;

CREATE TABLE budget_line (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_id BIGINT NOT NULL REFERENCES budget(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    allocation_amount NUMERIC(12, 2) NOT NULL,
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CHECK (allocation_amount >= 0),
    UNIQUE (budget_id, id),
    UNIQUE (budget_id, sort_order)
);

CREATE INDEX idx_budget_line_budget_id
ON budget_line(budget_id);

CREATE TABLE budget_line_category (
    budget_id BIGINT NOT NULL REFERENCES budget(id) ON DELETE CASCADE,
    budget_line_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES category(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (budget_line_id, category_id),
    FOREIGN KEY (budget_id, budget_line_id)
        REFERENCES budget_line(budget_id, id)
        ON DELETE CASCADE,
    UNIQUE (budget_id, category_id)
);

CREATE INDEX idx_budget_line_category_budget_id
ON budget_line_category(budget_id);

CREATE INDEX idx_budget_line_category_category_id
ON budget_line_category(category_id);

-- migrate:down
SET search_path TO transactions, public;

DROP INDEX IF EXISTS idx_budget_line_category_category_id;
DROP INDEX IF EXISTS idx_budget_line_category_budget_id;
DROP TABLE IF EXISTS budget_line_category;
DROP INDEX IF EXISTS idx_budget_line_budget_id;
DROP TABLE IF EXISTS budget_line;

DROP INDEX IF EXISTS idx_budget_user_period_start;
DROP INDEX IF EXISTS idx_budget_household_period_start;
DROP INDEX IF EXISTS idx_budget_user_period;
DROP INDEX IF EXISTS idx_budget_household_period;

ALTER TABLE budget DROP CONSTRAINT IF EXISTS chk_budget_valid_period;
ALTER TABLE budget DROP CONSTRAINT IF EXISTS chk_budget_exactly_one_owner;
ALTER TABLE budget DROP COLUMN IF EXISTS source_budget_id;
ALTER TABLE budget DROP COLUMN IF EXISTS period_end;
ALTER TABLE budget DROP COLUMN IF EXISTS period_start;
ALTER TABLE budget ADD COLUMN type VARCHAR(50) NOT NULL DEFAULT 'monthly';
