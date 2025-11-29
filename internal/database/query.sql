-- ******************* transaction *******************
-- READS

-- name: GetTransactionsByTransactionId :many
SELECT * FROM transaction
WHERE transaction_id = ANY(sqlc.arg(ids)::text[]);

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
        WHEN sqlc.arg(set_author_id)::bool THEN sqlc.arg(author_id)::int
        ELSE author_id
    END,
    budget_category_id = CASE
        WHEN sqlc.arg(set_budget_category_id)::bool THEN sqlc.narg(budget_category_id)::int
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
        WHEN sqlc.arg(set_household_id)::bool THEN sqlc.narg(household_id)::int
        ELSE household_id
    END
WHERE
    transaction_id = ANY(sqlc.arg(ids)::string[]) RETURNING *;
-- ******************* users *******************
-- READS

-- name: GetUserDetailsByDiscordId :one
SELECT sqlc.embed(users), sqlc.embed(household) FROM users
JOIN household_user on users.id = household_user.user_id
JOIN household on household_user.household_id = household.id
WHERE users.discord_id = $1;

-- name: GetUserByDiscordId :one
SELECT * FROM users WHERE discord_id = $1;

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

-- name: ListLlmMessagesBySessionId :many
SELECT
    sqlc.embed(m), sqlc.embed(u), sqlc.embed(h)
FROM
    llm_message m
JOIN
    users u on u.id = m.user_id
LEFT JOIN household_user hu on hu.user_id = u.id
LEFT JOIN household h on h.id = hu.household_id
WHERE
    m.session_id = $1
ORDER BY
    m.created_at ASC;