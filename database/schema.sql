-- TODO: ADD INDEXES FOR QUERIED COLUMNS

CREATE TABLE user (
    id INT PRIMARY KEY AUTO_INCREMENT,
    discord_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE household (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    UNIQUE (name),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE household_user (
    household_id INT,
    user_id INT,
    PRIMARY KEY (household_id, user_id),
    FOREIGN KEY (household_id) REFERENCES household(id),
    FOREIGN KEY (user_id) REFERENCES user(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE budget (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT,
    household_id INT,
    type VARCHAR(50) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user(id),
    FOREIGN KEY (household_id) REFERENCES household(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE budget_category (
    id INT PRIMARY KEY AUTO_INCREMENT,
    budget_id INT,
    category_name VARCHAR(255) NOT NULL,
    allocation FLOAT NOT NULL,
    FOREIGN KEY (budget_id) REFERENCES budget(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE transaction (
    id INT PRIMARY KEY AUTO_INCREMENT,
    amount FLOAT NOT NULL,
    paid_by INT NOT NULL,
    amount_owed FLOAT,
    budget_category_id INT,
    description VARCHAR(255),
    transaction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    transaction_id VARCHAR(255),
    transaction_type INT, -- 1=personal, 2=household

    -- for household
    owed_by INT,
    household_id INT,
    is_paid BOOLEAN,
    payment_date TIMESTAMP,

    UNIQUE(transaction_id),
    FOREIGN KEY (paid_by) REFERENCES user(id),
    FOREIGN KEY (owed_by) REFERENCES user(id),
    FOREIGN KEY (budget_category_id) REFERENCES budget_category(id),
    FOREIGN KEY (household_id) REFERENCES household(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
