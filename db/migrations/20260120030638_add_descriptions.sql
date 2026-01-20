-- migrate:up

-- Users Table
COMMENT ON TABLE users IS 'Stores identity information for individuals linked to Discord accounts.';
COMMENT ON COLUMN users.id IS 'Internal unique identifier for the user.';
COMMENT ON COLUMN users.discord_id IS 'The unique ID provided by the Discord API.';
COMMENT ON COLUMN users.name IS 'The display name or username of the Discord user.';
COMMENT ON COLUMN users.created_at IS 'Timestamp when the user record was first created.';
COMMENT ON COLUMN users.updated_at IS 'Timestamp of the most recent modification to the user record.';

-- Household Table
COMMENT ON TABLE household IS 'Groups users together into a shared financial unit, linked to a Discord server.';
COMMENT ON COLUMN household.id IS 'Internal unique identifier for the household.';
COMMENT ON COLUMN household.name IS 'A unique human-readable name for the household group.';
COMMENT ON COLUMN household.guild_id IS 'The Discord Guild (server) ID associated with this household.';
COMMENT ON COLUMN household.created_at IS 'Timestamp when the household was established.';
COMMENT ON COLUMN household.updated_at IS 'Timestamp of the last update to household metadata.';

-- Household_User Table
COMMENT ON TABLE household_user IS 'A join table representing the many-to-many relationship between users and households.';
COMMENT ON COLUMN household_user.household_id IS 'Reference to the household.';
COMMENT ON COLUMN household_user.user_id IS 'Reference to the user.';
COMMENT ON COLUMN household_user.created_at IS 'Timestamp when the user joined the household.';
COMMENT ON COLUMN household_user.updated_at IS 'Timestamp of the last change to this membership record.';

-- Budget Table
COMMENT ON TABLE budget IS 'Defines a financial plan, which can be assigned to either an individual or a household.';
COMMENT ON COLUMN budget.id IS 'Internal unique identifier for the budget.';
COMMENT ON COLUMN budget.user_id IS 'Optional reference to a specific user for personal budgets.';
COMMENT ON COLUMN budget.household_id IS 'Optional reference to a household for shared group budgets.';

-- Budget_Category Table
COMMENT ON TABLE budget_category IS 'Breaks down a budget into specific spending areas with allocated funds.';
COMMENT ON COLUMN budget_category.id IS 'Internal unique identifier for the budget category.';
COMMENT ON COLUMN budget_category.budget_id IS 'Reference to the parent budget.';
COMMENT ON COLUMN budget_category.category_name IS 'The label for the spending category (e.g., Groceries, Rent).';
COMMENT ON COLUMN budget_category.allocation IS 'The total amount of currency allocated to this category.';

-- Transaction Table
COMMENT ON TABLE transaction IS 'Records individual financial movements including amount, author, and categorization.';
COMMENT ON COLUMN transaction.id IS 'Internal unique identifier for the transaction.';
COMMENT ON COLUMN transaction.amount IS 'The numerical value of the transaction. If positive, it indicates an expense; if negative, income.';
COMMENT ON COLUMN transaction.author_id IS 'The user who created or is responsible for the transaction.';
COMMENT ON COLUMN transaction.budget_category_id IS 'Optional reference to link the transaction to a budget category.';
COMMENT ON COLUMN transaction.description IS 'A short summary of the transaction purpose.';
COMMENT ON COLUMN transaction.transaction_date IS 'The actual date and time the financial event occurred.';
COMMENT ON COLUMN transaction.transaction_id IS 'A unique hash or external identifier to prevent duplicate entries.';
COMMENT ON COLUMN transaction.household_id IS 'The household context in which this transaction took place.';
COMMENT ON COLUMN transaction.notes IS 'Extended details or commentary regarding the transaction.';

-- migrate:down

-- Users Table
COMMENT ON TABLE users IS NULL;
COMMENT ON COLUMN users.id IS NULL;
COMMENT ON COLUMN users.discord_id IS NULL;
COMMENT ON COLUMN users.name IS NULL;
COMMENT ON COLUMN users.created_at IS NULL;
COMMENT ON COLUMN users.updated_at IS NULL;

-- Household Table
COMMENT ON TABLE household IS NULL;
COMMENT ON COLUMN household.id IS NULL;
COMMENT ON COLUMN household.name IS NULL;
COMMENT ON COLUMN household.guild_id IS NULL;
COMMENT ON COLUMN household.created_at IS NULL;
COMMENT ON COLUMN household.updated_at IS NULL;

-- Household_User Table
COMMENT ON TABLE household_user IS NULL;
COMMENT ON COLUMN household_user.household_id IS NULL;
COMMENT ON COLUMN household_user.user_id IS NULL;
COMMENT ON COLUMN household_user.created_at IS NULL;
COMMENT ON COLUMN household_user.updated_at IS NULL;

-- Budget Table
COMMENT ON TABLE budget IS NULL;
COMMENT ON COLUMN budget.id IS NULL;
COMMENT ON COLUMN budget.user_id IS NULL;
COMMENT ON COLUMN budget.household_id IS NULL;
COMMENT ON COLUMN budget.type IS NULL;

-- Budget_Category Table
COMMENT ON TABLE budget_category IS NULL;
COMMENT ON COLUMN budget_category.id IS NULL;
COMMENT ON COLUMN budget_category.budget_id IS NULL;
COMMENT ON COLUMN budget_category.category_name IS NULL;
COMMENT ON COLUMN budget_category.allocation IS NULL;

-- Transaction Table
COMMENT ON TABLE transaction IS NULL;
COMMENT ON COLUMN transaction.id IS NULL;
COMMENT ON COLUMN transaction.amount IS NULL;
COMMENT ON COLUMN transaction.author_id IS NULL;
COMMENT ON COLUMN transaction.budget_category_id IS NULL;
COMMENT ON COLUMN transaction.description IS NULL;
COMMENT ON COLUMN transaction.transaction_date IS NULL;
COMMENT ON COLUMN transaction.transaction_id IS NULL;
COMMENT ON COLUMN transaction.household_id IS NULL;
COMMENT ON COLUMN transaction.notes IS NULL;