-- name: GetHouseholdTransaction :one
SELECT * FROM household_transaction
WHERE id = ? LIMIT 1;