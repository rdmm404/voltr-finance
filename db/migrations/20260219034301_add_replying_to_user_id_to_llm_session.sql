-- migrate:up
ALTER TABLE llm_session ADD COLUMN replying_to_user_id BIGINT REFERENCES users(id);

-- migrate:down
ALTER TABLE llm_session DROP COLUMN replying_to_user_id;
