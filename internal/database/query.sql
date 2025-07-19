-- name: ListTransactionsByHousehold :many
SELECT * FROM `transaction`
WHERE transaction_type=2 AND household_id = ?;

-- name: CreateTransaction :execresult
INSERT INTO `transaction`
SET
    amount = ?,
    is_paid = ?,
    amount_owed = ?,
    budget_category_id = ?,
    description = ?,
    transaction_date = ?,
    transaction_id = ?,
    transaction_type = ?,
    paid_by = ?;