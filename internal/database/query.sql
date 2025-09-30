-- ******************* transaction *******************
-- READS

-- name: ListTransactionsByHousehold :many
SELECT * FROM transaction
WHERE transaction_type=2 AND household_id = $1;

-- WRITES

-- name: CreateTransaction :one
INSERT INTO transaction
(
    amount,
    is_paid,
    amount_owed,
    budget_category_id,
    description,
    transaction_date,
    transaction_id,
    transaction_type,
    paid_by,
    household_id,
    notes
)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateTransactionById :one
UPDATE
    transaction
SET
    amount = CASE
        WHEN sqlc.arg(set_amount)::bool THEN sqlc.arg(amount)::real
        ELSE amount
    END,
    paid_by = CASE
        WHEN sqlc.arg(set_paid_by)::bool THEN sqlc.arg(paid_by)::int
        ELSE paid_by
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
    transaction_id = CASE
        WHEN sqlc.arg(set_transaction_id)::bool THEN sqlc.narg(transaction_id)::text
        ELSE transaction_id
    END,
    transaction_type = CASE
        WHEN sqlc.arg(set_transaction_type)::bool THEN sqlc.narg(transaction_type)::text
        ELSE transaction_type
    END,
    notes = CASE
        WHEN sqlc.arg(set_notes)::bool THEN sqlc.narg(notes)::text
        ELSE notes
    END,
    household_id = CASE
        WHEN sqlc.arg(set_household_id)::bool THEN sqlc.narg(household_id)::int
        ELSE household_id
    END,
    owed_by = CASE
        WHEN sqlc.arg(set_owed_by)::bool THEN sqlc.narg(owed_by)::int
        ELSE owed_by
    END,
    amount_owed = CASE
        WHEN sqlc.arg(set_amount_owed)::bool THEN sqlc.narg(amount_owed)::real
        ELSE amount_owed
    END,
    is_paid = CASE
        WHEN sqlc.arg(set_is_paid)::bool THEN sqlc.narg(is_paid)::bool
        ELSE is_paid
    END,
    payment_date = CASE
        WHEN sqlc.arg(set_payment_date)::bool THEN sqlc.narg(payment_date)::timestamp
        ELSE payment_date
    END
WHERE
    id = ANY(sqlc.arg(ids)::int[]) RETURNING *;
-- ******************* users *******************
-- READS

-- name: GetUserDetailsByDiscordId :one
SELECT sqlc.embed(users), sqlc.embed(household) FROM users
JOIN household_user on users.id = household_user.user_id
JOIN household on household_user.household_id = household.id
WHERE users.discord_id = $1;

-- ******************* LLM *******************
-- name: CreateLlmSession :one
INSERT INTO llm_session (user_id) VALUES ($1) RETURNING *;

-- name: CreateLlmMessage :one
INSERT INTO llm_message (session_id, role, contents) VALUES ($1, $2, $3) RETURNING *;
