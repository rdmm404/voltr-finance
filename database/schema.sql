-- TODO: ADD INDEXES FOR QUERIED COLUMNS
SET search_path = transactions;
ALTER DATABASE voltr_finance SET search_path TO transactions;
CREATE SCHEMA transactions;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- TODO add discord server id to link server to household
CREATE TABLE household (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- TODO add default owed amount
CREATE TABLE household_user (
    household_id INT,
    user_id INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (household_id, user_id),
    FOREIGN KEY (household_id) REFERENCES household(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE budget (
    id SERIAL PRIMARY KEY,
    user_id INT,
    household_id INT,
    type VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (household_id) REFERENCES household(id)
);

CREATE TABLE budget_category (
    id SERIAL PRIMARY KEY,
    budget_id INT,
    category_name VARCHAR(255) NOT NULL,
    allocation REAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (budget_id) REFERENCES budget(id)
);

CREATE TABLE transaction (
    id SERIAL PRIMARY KEY,
    amount REAL NOT NULL,
    paid_by INT NOT NULL,
    amount_owed REAL,
    budget_category_id INT,
    description VARCHAR(255),
    transaction_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    transaction_id VARCHAR(255) UNIQUE, -- hash
    transaction_type INT, -- 1=personal, 2=household
    notes TEXT,

    -- for household
    owed_by INT,
    household_id INT,
    is_paid BOOLEAN,
    payment_date TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (paid_by) REFERENCES users(id),
    FOREIGN KEY (owed_by) REFERENCES users(id),
    FOREIGN KEY (budget_category_id) REFERENCES budget_category(id),
    FOREIGN KEY (household_id) REFERENCES household(id)
);

-- LLM messages
-- TODO track token usage
CREATE TABLE llm_session (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    source_id VARCHAR(255) NOT NULL, -- discord channel id, NULL if dm
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE llm_message (
    id SERIAL PRIMARY KEY,
    session_id INT NOT NULL,
    parent_id INT,
    role VARCHAR(255) NOT NULL,
    contents JSONB NOT NULL,
    user_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES llm_session(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES llm_message(id)
);
