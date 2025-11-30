SET search_path = transactions;
ALTER DATABASE voltr_finance SET search_path TO transactions;
CREATE SCHEMA transactions;

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    discord_id VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_discord_id ON users(discord_id);

-- TODO add discord server id to link server to household
CREATE TABLE household (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE,
    guild_id VARCHAR UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_household_guild_id ON household(guild_id);

-- TODO add default owed amount
CREATE TABLE household_user (
    household_id BIGINT,
    user_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (household_id, user_id),
    FOREIGN KEY (household_id) REFERENCES household(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_household_user_user_id ON household_user(user_id);
CREATE INDEX idx_household_user_household_id ON household_user(household_id);

CREATE TABLE budget (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT,
    household_id BIGINT,
    type VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (household_id) REFERENCES household(id)
);

CREATE TABLE budget_category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_id BIGINT,
    category_name VARCHAR NOT NULL,
    allocation REAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (budget_id) REFERENCES budget(id)
);

CREATE TABLE transaction (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    amount REAL NOT NULL,
    author_id BIGINT NOT NULL,
    budget_category_id BIGINT,
    description VARCHAR,
    transaction_date TIMESTAMP WITH TIME ZONE NOT NULL,
    transaction_id VARCHAR UNIQUE NOT NULL, -- hash
    household_id BIGINT,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(id),
    FOREIGN KEY (budget_category_id) REFERENCES budget_category(id),
    FOREIGN KEY (household_id) REFERENCES household(id)
);
CREATE INDEX idx_transaction_household_id ON transaction(household_id);
CREATE INDEX idx_transaction_author_id ON transaction(author_id);
CREATE INDEX idx_transaction_budget_category_id ON transaction(budget_category_id);

-- LLM messages
-- TODO track token usage
CREATE TABLE llm_session (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    source_id VARCHAR NOT NULL, -- discord channel id, NULL if dm
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_llm_session_source_id ON llm_session(source_id);
CREATE INDEX idx_llm_session_user_id ON llm_session(user_id);

CREATE TABLE llm_message (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_id BIGINT NOT NULL,
    parent_id BIGINT,
    role VARCHAR NOT NULL,
    contents JSONB NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES llm_session(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES llm_message(id)
);
CREATE INDEX idx_llm_message_session_id ON llm_message(session_id);
CREATE INDEX idx_llm_message_created_at ON llm_message(created_at);
