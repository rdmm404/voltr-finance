-- name: ListTransactionsByHousehold :many
SELECT * FROM transactions.transaction
WHERE transaction_type=2 AND household_id = $1;

-- name: CreateTransaction :execresult
INSERT INTO transactions.transaction
(amount, is_paid, amount_owed, budget_category_id, description, transaction_date, transaction_id, transaction_type, paid_by)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8, $9);
