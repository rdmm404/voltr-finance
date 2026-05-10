-- ******************* category *******************
-- WRITES

-- name: CreateCategory :one
INSERT INTO category (code, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- READS

-- name: ListCategories :many
SELECT * FROM category
WHERE (sqlc.arg(include_inactive)::bool OR is_active)
ORDER BY name ASC, id ASC;

-- name: GetCategoryById :one
SELECT * FROM category
WHERE id = $1;

-- name: GetActiveCategoryById :one
SELECT * FROM category
WHERE id = $1 AND is_active;

-- name: GetCategoryByCode :one
SELECT * FROM category
WHERE code = $1;

-- name: GetActiveCategoryByCode :one
SELECT * FROM category
WHERE code = $1 AND is_active;

-- WRITES

-- name: UpdateCategory :one
UPDATE category
SET
    name = CASE
        WHEN sqlc.arg(set_name)::bool THEN sqlc.arg(name)::VARCHAR
        ELSE name
    END,
    description = CASE
        WHEN sqlc.arg(set_description)::bool THEN sqlc.narg(description)::TEXT
        ELSE description
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)::BIGINT
RETURNING *;

-- name: DeactivateCategory :one
UPDATE category
SET is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE code = $1
RETURNING *;

-- ******************* budget *******************
-- READS

-- name: GetHouseholdBudgetByPeriod :one
SELECT * FROM budget
WHERE household_id = sqlc.arg(household_id)::BIGINT
  AND user_id IS NULL
  AND period_start = sqlc.arg(period_start)::DATE
  AND period_end = sqlc.arg(period_end)::DATE;

-- name: GetUserBudgetByPeriod :one
SELECT * FROM budget
WHERE user_id = sqlc.arg(user_id)::BIGINT
  AND household_id IS NULL
  AND period_start = sqlc.arg(period_start)::DATE
  AND period_end = sqlc.arg(period_end)::DATE;

-- name: GetBudgetById :one
SELECT * FROM budget
WHERE id = sqlc.arg(id)::BIGINT;

-- name: GetLatestPriorHouseholdBudget :one
SELECT * FROM budget
WHERE household_id = sqlc.arg(household_id)::BIGINT
  AND user_id IS NULL
  AND period_start < sqlc.arg(period_start)::DATE
ORDER BY period_start DESC, id DESC
LIMIT 1;

-- name: GetLatestPriorUserBudget :one
SELECT * FROM budget
WHERE user_id = sqlc.arg(user_id)::BIGINT
  AND household_id IS NULL
  AND period_start < sqlc.arg(period_start)::DATE
ORDER BY period_start DESC, id DESC
LIMIT 1;

-- name: ListBudgetLines :many
SELECT * FROM budget_line
WHERE budget_id = sqlc.arg(budget_id)::BIGINT
ORDER BY sort_order ASC, id ASC;

-- name: ListBudgetLineCategories :many
SELECT
    blc.budget_id,
    blc.budget_line_id,
    blc.category_id,
    c.code AS category_code,
    c.name AS category_name
FROM budget_line_category blc
JOIN category c ON c.id = blc.category_id
WHERE blc.budget_id = sqlc.arg(budget_id)::BIGINT
ORDER BY blc.budget_line_id ASC, c.name ASC, c.id ASC;

-- name: GetBudgetLineById :one
SELECT * FROM budget_line
WHERE id = sqlc.arg(id)::BIGINT;

-- name: GetMaxBudgetLineSortOrder :one
SELECT COALESCE(MAX(sort_order), 0)::INTEGER AS sort_order
FROM budget_line
WHERE budget_id = sqlc.arg(budget_id)::BIGINT;

-- name: ListBudgetTransactions :many
SELECT
    t.category_id::BIGINT AS category_id,
    SUM(t.amount)::REAL AS actual_amount
FROM transaction t
WHERE t.deleted_at IS NULL
  AND t.category_id IS NOT NULL
  AND t.transaction_date >= sqlc.arg(period_start)::DATE
  AND t.transaction_date < (sqlc.arg(period_end)::DATE + INTERVAL '1 day')
  AND (
      (sqlc.narg(household_id)::BIGINT IS NOT NULL AND t.household_id = sqlc.narg(household_id)::BIGINT)
      OR
      (sqlc.narg(user_id)::BIGINT IS NOT NULL AND t.author_id = sqlc.narg(user_id)::BIGINT)
  )
GROUP BY t.category_id
ORDER BY t.category_id ASC;

-- name: SumUncategorizedBudgetTransactions :one
SELECT COALESCE(SUM(t.amount), 0)::REAL AS actual_amount
FROM transaction t
WHERE t.deleted_at IS NULL
  AND t.category_id IS NULL
  AND t.transaction_date >= sqlc.arg(period_start)::DATE
  AND t.transaction_date < (sqlc.arg(period_end)::DATE + INTERVAL '1 day')
  AND (
      (sqlc.narg(household_id)::BIGINT IS NOT NULL AND t.household_id = sqlc.narg(household_id)::BIGINT)
      OR
      (sqlc.narg(user_id)::BIGINT IS NOT NULL AND t.author_id = sqlc.narg(user_id)::BIGINT)
  );

-- WRITES

-- name: CreateHouseholdBudget :one
INSERT INTO budget (household_id, user_id, period_start, period_end, source_budget_id)
VALUES (
    sqlc.arg(household_id)::BIGINT,
    NULL,
    sqlc.arg(period_start)::DATE,
    sqlc.arg(period_end)::DATE,
    sqlc.narg(source_budget_id)::BIGINT
)
RETURNING *;

-- name: CreateUserBudget :one
INSERT INTO budget (household_id, user_id, period_start, period_end, source_budget_id)
VALUES (
    NULL,
    sqlc.arg(user_id)::BIGINT,
    sqlc.arg(period_start)::DATE,
    sqlc.arg(period_end)::DATE,
    sqlc.narg(source_budget_id)::BIGINT
)
RETURNING *;

-- name: CreateBudgetLine :one
INSERT INTO budget_line (budget_id, name, allocation_amount, sort_order)
VALUES (
    sqlc.arg(budget_id)::BIGINT,
    sqlc.arg(name)::VARCHAR,
    sqlc.arg(allocation_amount)::NUMERIC,
    sqlc.arg(sort_order)::INTEGER
)
RETURNING *;

-- name: UpdateBudgetLine :one
UPDATE budget_line
SET
    name = CASE
        WHEN sqlc.arg(set_name)::bool THEN sqlc.arg(name)::VARCHAR
        ELSE name
    END,
    allocation_amount = CASE
        WHEN sqlc.arg(set_allocation_amount)::bool THEN sqlc.arg(allocation_amount)::NUMERIC
        ELSE allocation_amount
    END,
    sort_order = CASE
        WHEN sqlc.arg(set_sort_order)::bool THEN sqlc.arg(sort_order)::INTEGER
        ELSE sort_order
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)::BIGINT
RETURNING *;

-- name: DeleteBudgetLine :exec
DELETE FROM budget_line
WHERE id = sqlc.arg(id)::BIGINT;

-- name: DeleteBudgetLineCategories :exec
DELETE FROM budget_line_category
WHERE budget_line_id = sqlc.arg(budget_line_id)::BIGINT;

-- name: CreateBudgetLineCategory :exec
INSERT INTO budget_line_category (budget_id, budget_line_id, category_id)
VALUES (
    sqlc.arg(budget_id)::BIGINT,
    sqlc.arg(budget_line_id)::BIGINT,
    sqlc.arg(category_id)::BIGINT
);

-- ******************* transaction *******************
-- READS

-- name: GetTransactionById :one
SELECT * FROM transaction
WHERE id = $1;

-- name: GetTransactionsById :many
SELECT * FROM transaction
WHERE id = ANY(sqlc.arg(ids)::BIGINT[]);

-- name: GetTransactionByIdActive :one
SELECT * FROM transaction
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTransactionsByIdActive :many
SELECT * FROM transaction
WHERE id = ANY(sqlc.arg(ids)::BIGINT[])
  AND deleted_at IS NULL;

-- name: GetTransactionsByIdWithDetails :many
SELECT
    sqlc.embed(t),
    u.id AS author_id,
    u.name AS author_name,
    h.id AS household_id,
    h.name AS household_name,
    c.id AS category_id,
    c.code AS category_code,
    c.name AS category_name
FROM transaction t
JOIN users u ON u.id = t.author_id
LEFT JOIN household h ON h.id = t.household_id
LEFT JOIN category c ON c.id = t.category_id
WHERE t.id = ANY(sqlc.arg(ids)::BIGINT[])
  AND (sqlc.arg(include_deleted)::bool OR t.deleted_at IS NULL)
ORDER BY array_position(sqlc.arg(ids)::BIGINT[], t.id);

-- name: GetIdByTransactionId :one
SELECT id FROM transaction
WHERE transaction_id = $1;


-- name: ListTransactionsByHousehold :many
SELECT * FROM transaction
WHERE transaction_type=2 AND household_id = $1;

-- name: ListTransactions :many
SELECT
    sqlc.embed(t),
    u.id AS author_id,
    u.name AS author_name,
    h.id AS household_id,
    h.name AS household_name,
    c.id AS category_id,
    c.code AS category_code,
    c.name AS category_name
FROM transaction t
JOIN users u ON u.id = t.author_id
LEFT JOIN household h ON h.id = t.household_id
LEFT JOIN category c ON c.id = t.category_id
WHERE
    (NOT sqlc.arg(only_deleted)::bool OR t.deleted_at IS NOT NULL)
    AND (sqlc.arg(include_deleted)::bool OR sqlc.arg(only_deleted)::bool OR t.deleted_at IS NULL)
    AND (sqlc.narg(author_id)::BIGINT IS NULL OR t.author_id = sqlc.narg(author_id)::BIGINT)
    AND (sqlc.narg(household_id)::BIGINT IS NULL OR t.household_id = sqlc.narg(household_id)::BIGINT)
    AND (sqlc.narg(from_date)::TIMESTAMPTZ IS NULL OR t.transaction_date >= sqlc.narg(from_date)::TIMESTAMPTZ)
    AND (sqlc.narg(to_date)::TIMESTAMPTZ IS NULL OR t.transaction_date <= sqlc.narg(to_date)::TIMESTAMPTZ)
    AND (
        sqlc.narg(search)::TEXT IS NULL
        OR t.description ILIKE '%' || sqlc.narg(search)::TEXT || '%'
        OR t.notes ILIKE '%' || sqlc.narg(search)::TEXT || '%'
    )
ORDER BY
    CASE WHEN sqlc.arg(sort)::TEXT = 'transaction_date' AND sqlc.arg(sort_order)::TEXT = 'asc' THEN t.transaction_date END ASC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'transaction_date' AND sqlc.arg(sort_order)::TEXT = 'desc' THEN t.transaction_date END DESC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'created_at' AND sqlc.arg(sort_order)::TEXT = 'asc' THEN t.created_at END ASC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'created_at' AND sqlc.arg(sort_order)::TEXT = 'desc' THEN t.created_at END DESC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'amount' AND sqlc.arg(sort_order)::TEXT = 'asc' THEN t.amount END ASC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'amount' AND sqlc.arg(sort_order)::TEXT = 'desc' THEN t.amount END DESC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'id' AND sqlc.arg(sort_order)::TEXT = 'asc' THEN t.id END ASC,
    CASE WHEN sqlc.arg(sort)::TEXT = 'id' AND sqlc.arg(sort_order)::TEXT = 'desc' THEN t.id END DESC,
    t.transaction_date DESC,
    t.id DESC
LIMIT sqlc.arg(result_limit)::INT
OFFSET sqlc.arg(result_offset)::INT;

-- WRITES

-- name: CreateTransaction :one
INSERT INTO transaction
(
    amount,
    category_id,
    description,
    transaction_date,
    transaction_id,
    author_id,
    household_id,
    notes
)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTransactionById :one
UPDATE
    transaction
SET
    amount = CASE
        WHEN sqlc.arg(set_amount)::bool THEN sqlc.arg(amount)::real
        ELSE amount
    END,
    author_id = CASE
        WHEN sqlc.arg(set_author_id)::bool THEN sqlc.arg(author_id)::BIGINT
        ELSE author_id
    END,
    category_id = CASE
        WHEN sqlc.arg(set_category_id)::bool THEN sqlc.narg(category_id)::BIGINT
        ELSE category_id
    END,
    description = CASE
        WHEN sqlc.arg(set_description)::bool THEN sqlc.narg(description)::text
        ELSE description
    END,
    transaction_date = CASE
        WHEN sqlc.arg(set_transaction_date)::bool THEN sqlc.narg(transaction_date)::timestamp
        ELSE transaction_date
    END,
    notes = CASE
        WHEN sqlc.arg(set_notes)::bool THEN sqlc.narg(notes)::text
        ELSE notes
    END,
    household_id = CASE
        WHEN sqlc.arg(set_household_id)::bool THEN sqlc.narg(household_id)::BIGINT
        ELSE household_id
    END,
    transaction_id = $2
WHERE
    id = $1 RETURNING *;

-- name: SoftDeleteTransactionsById :many
UPDATE transaction
SET
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by_user_id = sqlc.arg(deleted_by_user_id)::BIGINT,
    delete_reason = sqlc.narg(delete_reason)::TEXT,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ANY(sqlc.arg(ids)::BIGINT[])
  AND deleted_at IS NULL
RETURNING *;

-- name: RestoreTransactionsById :many
UPDATE transaction
SET
    deleted_at = NULL,
    deleted_by_user_id = NULL,
    delete_reason = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ANY(sqlc.arg(ids)::BIGINT[])
  AND deleted_at IS NOT NULL
RETURNING *;

-- ******************* users *******************
-- READS

-- name: GetUserDetailsByDiscordId :one
SELECT sqlc.embed(users), sqlc.embed(household) FROM users
JOIN household_user on users.id = household_user.user_id
JOIN household on household_user.household_id = household.id
WHERE users.discord_id = $1;

-- name: GetUserByDiscordId :one
SELECT * FROM users WHERE discord_id = $1;

-- name: GetUserById :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByTelegramId :one
SELECT * FROM users WHERE telegram_id = $1;

-- name: GetUserByPhoneNumber :one
SELECT * FROM users WHERE phone_number = $1;

-- name: GetUserByWhatsappId :one
SELECT * FROM users WHERE whatsapp_id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY name ASC, id ASC;

-- name: CreateUser :one
INSERT INTO users (discord_id, telegram_id, phone_number, whatsapp_id, name)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET
    discord_id = CASE
        WHEN sqlc.arg(set_discord_id)::bool THEN sqlc.narg(discord_id)::VARCHAR
        ELSE discord_id
    END,
    telegram_id = CASE
        WHEN sqlc.arg(set_telegram_id)::bool THEN sqlc.narg(telegram_id)::VARCHAR
        ELSE telegram_id
    END,
    phone_number = CASE
        WHEN sqlc.arg(set_phone_number)::bool THEN sqlc.narg(phone_number)::VARCHAR
        ELSE phone_number
    END,
    whatsapp_id = CASE
        WHEN sqlc.arg(set_whatsapp_id)::bool THEN sqlc.narg(whatsapp_id)::VARCHAR
        ELSE whatsapp_id
    END,
    name = CASE
        WHEN sqlc.arg(set_name)::bool THEN sqlc.arg(name)::VARCHAR
        ELSE name
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)::BIGINT
RETURNING *;

-- name: GetUserByDiscordAndHouseholdId :one
SELECT u.* FROM users u
JOIN household_user hu on hu.user_id = u.id
WHERE discord_id = $1 and hu.household_id = $2;

-- name: GetHouseholdById :one
SELECT * FROM household WHERE id = $1;

-- name: GetHouseholdByGuildId :one
SELECT * from household where guild_id = $1;

-- name: GetHouseholdByName :one
SELECT * FROM household WHERE name = $1;

-- name: ListHouseholds :many
SELECT * FROM household
ORDER BY name ASC, id ASC;

-- name: GetHouseholdUsers :many
SELECT u.* FROM users u
JOIN household_user hu on hu.user_id = u.id
WHERE hu.household_id = $1;

-- ******************* LLM *******************
-- Session
-- name: CreateLlmSession :one
INSERT INTO
    llm_session (user_id, source_id)
VALUES
    ($1, $2) RETURNING *;

-- name: GetActiveSessionBySourceId :one
SELECT
    *
FROM
    llm_session
WHERE
    source_id = $1
ORDER BY created_at DESC;

-- Messages
-- name: CreateLlmMessage :one
INSERT INTO
    llm_message (session_id, user_id, role, contents, parent_id)
VALUES
    ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateMessageContents :exec
UPDATE llm_message SET contents = $2 WHERE id = $1;


-- name: ListLlmMessagesBySessionId :many
SELECT
    sqlc.embed(m), sqlc.embed(u)
FROM
    llm_message m
JOIN
    users u on u.id = m.user_id
WHERE
    m.session_id = $1
ORDER BY
    m.created_at ASC;


-- ******************* sql metadata *******************

-- name: GetTableAndColumnMetadata :many
SELECT
  t.table_schema::text AS schema_name,
  t.table_name::text AS table_name,
  COALESCE(obj_description(pgc.oid, 'pg_class'), '')::text AS table_description,
  c.column_name::text AS column_name,
  c.data_type::text AS data_type,
  COALESCE(col_description(pgc.oid, c.ordinal_position), '')::text AS column_description,
  (c.is_nullable = 'YES')::boolean AS is_nullable,
  -- Check if part of any index
  EXISTS (
    SELECT 1
    FROM pg_catalog.pg_index i
    WHERE i.indrelid = pgc.oid
      AND c.ordinal_position = ANY(i.indkey)
  )::boolean AS is_indexed,
  -- Check if part of a unique index
  EXISTS (
    SELECT 1
    FROM pg_catalog.pg_index i
    WHERE i.indrelid = pgc.oid
      AND i.indisunique = true
      AND c.ordinal_position = ANY(i.indkey)
  )::boolean AS is_unique,
  -- 1. Foreign Key in format table.column
  COALESCE(
    (
      SELECT ta.relname || '.' || fa.attname
      FROM pg_catalog.pg_constraint con
      JOIN pg_catalog.pg_class ta ON con.confrelid = ta.oid
      JOIN pg_catalog.pg_attribute fa ON fa.attrelid = con.confrelid AND fa.attnum = ANY(con.confkey)
      WHERE con.conrelid = pgc.oid
        AND con.contype = 'f'
        AND c.ordinal_position = ANY(con.conkey)
      LIMIT 1
    ),
    ''
  )::text AS foreign_key_target,
  -- 3. Primary Key Identity
  EXISTS (
    SELECT 1
    FROM pg_catalog.pg_constraint con
    WHERE con.conrelid = pgc.oid
      AND con.contype = 'p'
      AND c.ordinal_position = ANY(con.conkey)
  )::boolean AS is_primary_key
FROM
  information_schema.tables t
  JOIN pg_catalog.pg_class pgc ON t.table_name = pgc.relname
  JOIN pg_catalog.pg_namespace pgn ON pgc.relnamespace = pgn.oid
  AND t.table_schema = pgn.nspname
  JOIN information_schema.columns c ON t.table_name = c.table_name
  AND t.table_schema = c.table_schema
WHERE
  t.table_schema = 'transactions'
  AND t.table_name = ANY(sqlc.arg(table_names)::TEXT[])
ORDER BY
  t.table_name,
  c.ordinal_position;
