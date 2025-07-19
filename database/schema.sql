CREATE TABLE user (
    id INT PRIMARY KEY AUTO_INCREMENT,
    discord_id VARCHAR(255) UNIQUE,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE household (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    UNIQUE (name)
);

CREATE TABLE household_user (
    household_id INT,
    user_id INT,
    PRIMARY KEY (household_id, user_id),
    FOREIGN KEY (household_id) REFERENCES household(id),
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE budget (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT,
    household_id INT,
    type VARCHAR(50) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user(id),
    FOREIGN KEY (household_id) REFERENCES household(id)
);

CREATE TABLE budget_category (
    id INT PRIMARY KEY AUTO_INCREMENT,
    budget_id INT,
    category_name VARCHAR(255) NOT NULL, -- TODO separate into new table if needed
    allocation FLOAT NOT NULL,
    FOREIGN KEY (budget_id) REFERENCES budget(id)
);

CREATE TABLE household_transaction (
    id INT PRIMARY KEY AUTO_INCREMENT,
    paid_by INT,
    amount FLOAT NOT NULL,
    is_paid BOOLEAN NOT NULL,
    amount_owed FLOAT,
    budget_category_id INT,
    household_id INT,
    FOREIGN KEY (paid_by) REFERENCES user(id),
    FOREIGN KEY (budget_category_id) REFERENCES budget_category(id),
    FOREIGN KEY (household_id) REFERENCES household(id)
);

CREATE TABLE personal_transaction (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT,
    budget_category_id INT,
    FOREIGN KEY (user_id) REFERENCES user(id),
    FOREIGN KEY (budget_category_id) REFERENCES budget_category(id)
);