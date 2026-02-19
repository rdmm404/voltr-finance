-- ******************* transaction *******************
-- READS

-- name: GetTransactionById :one
SELECT * FROM transaction
WHERE id = $1;

-- name: GetTransactionsById :many
SELECT * FROM transaction
WHERE id = ANY(sqlc.arg(ids)::BIGINT[]);

-- name: GetIdByTransactionId :one
SELECT id FROM transaction
WHERE transaction_id = $1;


-- name: ListTransactionsByHousehold :many
SELECT * FROM transaction
WHERE transaction_type=2 AND household_id = $1;

-- WRITES

-- name: CreateTransaction :one
INSERT INTO transaction
(
    amount,
    budget_category_id,
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
    budget_category_id = CASE
        WHEN sqlc.arg(set_budget_category_id)::bool THEN sqlc.narg(budget_category_id)::BIGINT
        ELSE budget_category_id
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
-- ******************* users *******************
-- READS

-- name: GetUserDetailsByDiscordId :one
SELECT sqlc.embed(users), sqlc.embed(household) FROM users
JOIN household_user on users.id = household_user.user_id
JOIN household on household_user.household_id = household.id
WHERE users.discord_id = $1;

-- name: GetUserByDiscordId :one
SELECT * FROM users WHERE discord_id = $1;

-- name: GetUserByDiscordAndHouseholdId :one
SELECT u.* FROM users u
JOIN household_user hu on hu.user_id = u.id
WHERE discord_id = $1 and hu.household_id = $2;

-- name: GetHouseholdByGuildId :one
SELECT * from household where guild_id = $1;

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

-- name: GetAndLockActiveSessionBySourceId :one
SELECT FOR UPDATE
    *
FROM
    llm_session
WHERE
    source_id = $1
ORDER BY created_at DESC
LIMIT 1;


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