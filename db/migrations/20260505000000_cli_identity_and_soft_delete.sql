-- migrate:up
ALTER TABLE users ALTER COLUMN discord_id DROP NOT NULL;
ALTER TABLE users ADD COLUMN telegram_id VARCHAR;
ALTER TABLE users ADD COLUMN phone_number VARCHAR;
ALTER TABLE users ADD COLUMN whatsapp_id VARCHAR;

CREATE UNIQUE INDEX idx_users_discord_id_unique_not_null ON users(discord_id) WHERE discord_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_telegram_id_unique_not_null ON users(telegram_id) WHERE telegram_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_phone_number_unique_not_null ON users(phone_number) WHERE phone_number IS NOT NULL;
CREATE UNIQUE INDEX idx_users_whatsapp_id_unique_not_null ON users(whatsapp_id) WHERE whatsapp_id IS NOT NULL;

ALTER TABLE transaction ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE transaction ADD COLUMN deleted_by_user_id BIGINT REFERENCES users(id);
ALTER TABLE transaction ADD COLUMN delete_reason TEXT;
CREATE INDEX idx_transaction_deleted_at ON transaction(deleted_at);

-- migrate:down
DROP INDEX IF EXISTS idx_transaction_deleted_at;
ALTER TABLE transaction DROP COLUMN IF EXISTS delete_reason;
ALTER TABLE transaction DROP COLUMN IF EXISTS deleted_by_user_id;
ALTER TABLE transaction DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_users_whatsapp_id_unique_not_null;
DROP INDEX IF EXISTS idx_users_phone_number_unique_not_null;
DROP INDEX IF EXISTS idx_users_telegram_id_unique_not_null;
DROP INDEX IF EXISTS idx_users_discord_id_unique_not_null;
ALTER TABLE users DROP COLUMN IF EXISTS whatsapp_id;
ALTER TABLE users DROP COLUMN IF EXISTS phone_number;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_id;
ALTER TABLE users ALTER COLUMN discord_id SET NOT NULL;
