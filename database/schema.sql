-- TODO: ADD INDEXES FOR QUERIED COLUMNS
CREATE SCHEMA transactions;

CREATE TABLE transactions."user" (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions.household (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions.household_user (
    household_id INT,
    user_id INT,
    PRIMARY KEY (household_id, user_id),
    FOREIGN KEY (household_id) REFERENCES transactions.household(id),
    FOREIGN KEY (user_id) REFERENCES transactions."user"(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions.budget (
    id SERIAL PRIMARY KEY,
    user_id INT,
    household_id INT,
    type VARCHAR(50) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES transactions."user"(id),
    FOREIGN KEY (household_id) REFERENCES transactions.household(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions.budget_category (
    id SERIAL PRIMARY KEY,
    budget_id INT,
    category_name VARCHAR(255) NOT NULL,
    allocation REAL NOT NULL,
    FOREIGN KEY (budget_id) REFERENCES transactions.budget(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions.transaction (
    id SERIAL PRIMARY KEY,
    amount REAL NOT NULL,
    paid_by INT NOT NULL,
    amount_owed REAL,
    budget_category_id INT,
    description VARCHAR(255),
    transaction_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    transaction_id VARCHAR(255) UNIQUE,
    transaction_type INT, -- 1=personal, 2=household

    -- for household
    owed_by INT,
    household_id INT,
    is_paid BOOLEAN,
    payment_date TIMESTAMP WITH TIME ZONE,

    FOREIGN KEY (paid_by) REFERENCES transactions."user"(id),
    FOREIGN KEY (owed_by) REFERENCES transactions."user"(id),
    FOREIGN KEY (budget_category_id) REFERENCES transactions.budget_category(id),
    FOREIGN KEY (household_id) REFERENCES transactions.household(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);