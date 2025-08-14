-- name: ListTransactionsByHousehold :many
SELECT * FROM transaction
WHERE transaction_type=2 AND household_id = $1;

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

-- name: GetUserDetailsByDiscordId :one
SELECT sqlc.embed(users), sqlc.embed(household) FROM users
JOIN household_user on users.id = household_user.user_id
JOIN household on household_user.household_id = household.id
WHERE users.discord_id = $1;
