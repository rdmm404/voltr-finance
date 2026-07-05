\restrict dbmate

-- Dumped from database version 18.1 (Debian 18.1-1.pgdg13+2)
-- Dumped by pg_dump version 18.1

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: transactions; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA transactions;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: budget; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.budget (
    id bigint NOT NULL,
    user_id bigint,
    household_id bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    period_start date NOT NULL,
    period_end date NOT NULL,
    source_budget_id bigint,
    CONSTRAINT chk_budget_exactly_one_owner CHECK ((((household_id IS NOT NULL) AND (user_id IS NULL)) OR ((household_id IS NULL) AND (user_id IS NOT NULL)))),
    CONSTRAINT chk_budget_valid_period CHECK ((period_end >= period_start))
);


--
-- Name: TABLE budget; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON TABLE transactions.budget IS 'Defines a financial plan, which can be assigned to either an individual or a household.';


--
-- Name: COLUMN budget.id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.budget.id IS 'Internal unique identifier for the budget.';


--
-- Name: COLUMN budget.user_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.budget.user_id IS 'Optional reference to a specific user for personal budgets.';


--
-- Name: COLUMN budget.household_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.budget.household_id IS 'Optional reference to a household for shared group budgets.';


--
-- Name: budget_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.budget ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.budget_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: budget_line; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.budget_line (
    id bigint NOT NULL,
    budget_id bigint NOT NULL,
    name character varying NOT NULL,
    allocation_amount numeric(12,2) NOT NULL,
    sort_order integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT budget_line_allocation_amount_check CHECK ((allocation_amount >= (0)::numeric))
);


--
-- Name: budget_line_category; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.budget_line_category (
    budget_id bigint NOT NULL,
    budget_line_id bigint NOT NULL,
    category_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: budget_line_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.budget_line ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.budget_line_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: category; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.category (
    id bigint NOT NULL,
    code character varying NOT NULL,
    name character varying NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: category_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.category ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.category_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: household; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.household (
    id bigint NOT NULL,
    name character varying NOT NULL,
    guild_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: TABLE household; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON TABLE transactions.household IS 'Groups users together into a shared financial unit, linked to a Discord server.';


--
-- Name: COLUMN household.id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household.id IS 'Internal unique identifier for the household.';


--
-- Name: COLUMN household.name; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household.name IS 'A unique human-readable name for the household group.';


--
-- Name: COLUMN household.guild_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household.guild_id IS 'The Discord Guild (server) ID associated with this household.';


--
-- Name: COLUMN household.created_at; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household.created_at IS 'Timestamp when the household was established.';


--
-- Name: COLUMN household.updated_at; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household.updated_at IS 'Timestamp of the last update to household metadata.';


--
-- Name: household_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.household ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.household_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: household_user; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.household_user (
    household_id bigint NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: TABLE household_user; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON TABLE transactions.household_user IS 'A join table representing the many-to-many relationship between users and households.';


--
-- Name: COLUMN household_user.household_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household_user.household_id IS 'Reference to the household.';


--
-- Name: COLUMN household_user.user_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household_user.user_id IS 'Reference to the user.';


--
-- Name: COLUMN household_user.created_at; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household_user.created_at IS 'Timestamp when the user joined the household.';


--
-- Name: COLUMN household_user.updated_at; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.household_user.updated_at IS 'Timestamp of the last change to this membership record.';


--
-- Name: llm_message; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.llm_message (
    id bigint NOT NULL,
    session_id bigint NOT NULL,
    parent_id bigint,
    role character varying NOT NULL,
    contents jsonb NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: llm_message_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.llm_message ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.llm_message_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: llm_session; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.llm_session (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    source_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: llm_session_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.llm_session ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.llm_session_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: schema_migrations; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: transaction; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.transaction (
    id bigint NOT NULL,
    amount real NOT NULL,
    author_id bigint NOT NULL,
    description character varying,
    transaction_date timestamp with time zone NOT NULL,
    transaction_id character varying NOT NULL,
    household_id bigint,
    notes text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone,
    deleted_by_user_id bigint,
    delete_reason text,
    category_id bigint
);


--
-- Name: TABLE transaction; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON TABLE transactions.transaction IS 'Records individual financial movements including amount, author, and categorization.';


--
-- Name: COLUMN transaction.id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.id IS 'Internal unique identifier for the transaction.';


--
-- Name: COLUMN transaction.amount; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.amount IS 'The numerical value of the transaction. If positive, it indicates an expense; if negative, income.';


--
-- Name: COLUMN transaction.author_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.author_id IS 'The user who created or is responsible for the transaction.';


--
-- Name: COLUMN transaction.description; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.description IS 'A short summary of the transaction purpose.';


--
-- Name: COLUMN transaction.transaction_date; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.transaction_date IS 'The actual date and time the financial event occurred.';


--
-- Name: COLUMN transaction.transaction_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.transaction_id IS 'A unique hash or external identifier to prevent duplicate entries.';


--
-- Name: COLUMN transaction.household_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.household_id IS 'The household context in which this transaction took place.';


--
-- Name: COLUMN transaction.notes; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.transaction.notes IS 'Extended details or commentary regarding the transaction.';


--
-- Name: transaction_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.transaction ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.transaction_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: users; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.users (
    id bigint NOT NULL,
    discord_id character varying,
    name character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    telegram_id character varying,
    phone_number character varying,
    whatsapp_id character varying
);


--
-- Name: TABLE users; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON TABLE transactions.users IS 'Stores identity information for individuals linked to Discord accounts.';


--
-- Name: COLUMN users.id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.users.id IS 'Internal unique identifier for the user.';


--
-- Name: COLUMN users.discord_id; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.users.discord_id IS 'The unique ID provided by the Discord API.';


--
-- Name: COLUMN users.name; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.users.name IS 'The display name or username of the Discord user.';


--
-- Name: COLUMN users.created_at; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.users.created_at IS 'Timestamp when the user record was first created.';


--
-- Name: COLUMN users.updated_at; Type: COMMENT; Schema: transactions; Owner: -
--

COMMENT ON COLUMN transactions.users.updated_at IS 'Timestamp of the most recent modification to the user record.';


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.users ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: budget_line budget_line_budget_id_id_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line
    ADD CONSTRAINT budget_line_budget_id_id_key UNIQUE (budget_id, id);


--
-- Name: budget_line budget_line_budget_id_sort_order_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line
    ADD CONSTRAINT budget_line_budget_id_sort_order_key UNIQUE (budget_id, sort_order);


--
-- Name: budget_line_category budget_line_category_budget_id_category_id_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line_category
    ADD CONSTRAINT budget_line_category_budget_id_category_id_key UNIQUE (budget_id, category_id);


--
-- Name: budget_line_category budget_line_category_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line_category
    ADD CONSTRAINT budget_line_category_pkey PRIMARY KEY (budget_line_id, category_id);


--
-- Name: budget_line budget_line_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line
    ADD CONSTRAINT budget_line_pkey PRIMARY KEY (id);


--
-- Name: budget budget_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget
    ADD CONSTRAINT budget_pkey PRIMARY KEY (id);


--
-- Name: category category_code_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.category
    ADD CONSTRAINT category_code_key UNIQUE (code);


--
-- Name: category category_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.category
    ADD CONSTRAINT category_pkey PRIMARY KEY (id);


--
-- Name: household household_guild_id_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.household
    ADD CONSTRAINT household_guild_id_key UNIQUE (guild_id);


--
-- Name: household household_name_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.household
    ADD CONSTRAINT household_name_key UNIQUE (name);


--
-- Name: household household_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.household
    ADD CONSTRAINT household_pkey PRIMARY KEY (id);


--
-- Name: household_user household_user_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.household_user
    ADD CONSTRAINT household_user_pkey PRIMARY KEY (household_id, user_id);


--
-- Name: llm_message llm_message_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.llm_message
    ADD CONSTRAINT llm_message_pkey PRIMARY KEY (id);


--
-- Name: llm_session llm_session_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.llm_session
    ADD CONSTRAINT llm_session_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: transaction transaction_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_pkey PRIMARY KEY (id);


--
-- Name: transaction transaction_transaction_id_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_transaction_id_key UNIQUE (transaction_id);


--
-- Name: users users_discord_id_key; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.users
    ADD CONSTRAINT users_discord_id_key UNIQUE (discord_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_budget_household_period; Type: INDEX; Schema: transactions; Owner: -
--

CREATE UNIQUE INDEX idx_budget_household_period ON transactions.budget USING btree (household_id, period_start, period_end) WHERE (household_id IS NOT NULL);


--
-- Name: idx_budget_household_period_start; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_budget_household_period_start ON transactions.budget USING btree (household_id, period_start) WHERE (household_id IS NOT NULL);


--
-- Name: idx_budget_line_budget_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_budget_line_budget_id ON transactions.budget_line USING btree (budget_id);


--
-- Name: idx_budget_line_category_budget_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_budget_line_category_budget_id ON transactions.budget_line_category USING btree (budget_id);


--
-- Name: idx_budget_line_category_category_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_budget_line_category_category_id ON transactions.budget_line_category USING btree (category_id);


--
-- Name: idx_budget_user_period; Type: INDEX; Schema: transactions; Owner: -
--

CREATE UNIQUE INDEX idx_budget_user_period ON transactions.budget USING btree (user_id, period_start, period_end) WHERE (user_id IS NOT NULL);


--
-- Name: idx_budget_user_period_start; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_budget_user_period_start ON transactions.budget USING btree (user_id, period_start) WHERE (user_id IS NOT NULL);


--
-- Name: idx_category_code; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_category_code ON transactions.category USING btree (code);


--
-- Name: idx_category_is_active; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_category_is_active ON transactions.category USING btree (is_active);


--
-- Name: idx_household_guild_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_household_guild_id ON transactions.household USING btree (guild_id);


--
-- Name: idx_household_user_household_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_household_user_household_id ON transactions.household_user USING btree (household_id);


--
-- Name: idx_household_user_user_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_household_user_user_id ON transactions.household_user USING btree (user_id);


--
-- Name: idx_llm_message_created_at; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_llm_message_created_at ON transactions.llm_message USING btree (created_at);


--
-- Name: idx_llm_message_session_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_llm_message_session_id ON transactions.llm_message USING btree (session_id);


--
-- Name: idx_llm_session_source_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_llm_session_source_id ON transactions.llm_session USING btree (source_id);


--
-- Name: idx_llm_session_user_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_llm_session_user_id ON transactions.llm_session USING btree (user_id);


--
-- Name: idx_transaction_author_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_transaction_author_id ON transactions.transaction USING btree (author_id);


--
-- Name: idx_transaction_category_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_transaction_category_id ON transactions.transaction USING btree (category_id);


--
-- Name: idx_transaction_deleted_at; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_transaction_deleted_at ON transactions.transaction USING btree (deleted_at);


--
-- Name: idx_transaction_household_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_transaction_household_id ON transactions.transaction USING btree (household_id);


--
-- Name: idx_users_discord_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_users_discord_id ON transactions.users USING btree (discord_id);


--
-- Name: idx_users_discord_id_unique_not_null; Type: INDEX; Schema: transactions; Owner: -
--

CREATE UNIQUE INDEX idx_users_discord_id_unique_not_null ON transactions.users USING btree (discord_id) WHERE (discord_id IS NOT NULL);


--
-- Name: idx_users_phone_number_unique_not_null; Type: INDEX; Schema: transactions; Owner: -
--

CREATE UNIQUE INDEX idx_users_phone_number_unique_not_null ON transactions.users USING btree (phone_number) WHERE (phone_number IS NOT NULL);


--
-- Name: idx_users_telegram_id_unique_not_null; Type: INDEX; Schema: transactions; Owner: -
--

CREATE UNIQUE INDEX idx_users_telegram_id_unique_not_null ON transactions.users USING btree (telegram_id) WHERE (telegram_id IS NOT NULL);


--
-- Name: idx_users_whatsapp_id_unique_not_null; Type: INDEX; Schema: transactions; Owner: -
--

CREATE UNIQUE INDEX idx_users_whatsapp_id_unique_not_null ON transactions.users USING btree (whatsapp_id) WHERE (whatsapp_id IS NOT NULL);


--
-- Name: budget budget_household_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget
    ADD CONSTRAINT budget_household_id_fkey FOREIGN KEY (household_id) REFERENCES transactions.household(id);


--
-- Name: budget_line budget_line_budget_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line
    ADD CONSTRAINT budget_line_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES transactions.budget(id) ON DELETE CASCADE;


--
-- Name: budget_line_category budget_line_category_budget_id_budget_line_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line_category
    ADD CONSTRAINT budget_line_category_budget_id_budget_line_id_fkey FOREIGN KEY (budget_id, budget_line_id) REFERENCES transactions.budget_line(budget_id, id) ON DELETE CASCADE;


--
-- Name: budget_line_category budget_line_category_budget_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line_category
    ADD CONSTRAINT budget_line_category_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES transactions.budget(id) ON DELETE CASCADE;


--
-- Name: budget_line_category budget_line_category_category_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_line_category
    ADD CONSTRAINT budget_line_category_category_id_fkey FOREIGN KEY (category_id) REFERENCES transactions.category(id);


--
-- Name: budget budget_source_budget_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget
    ADD CONSTRAINT budget_source_budget_id_fkey FOREIGN KEY (source_budget_id) REFERENCES transactions.budget(id);


--
-- Name: budget budget_user_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget
    ADD CONSTRAINT budget_user_id_fkey FOREIGN KEY (user_id) REFERENCES transactions.users(id);


--
-- Name: household_user household_user_household_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.household_user
    ADD CONSTRAINT household_user_household_id_fkey FOREIGN KEY (household_id) REFERENCES transactions.household(id);


--
-- Name: household_user household_user_user_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.household_user
    ADD CONSTRAINT household_user_user_id_fkey FOREIGN KEY (user_id) REFERENCES transactions.users(id);


--
-- Name: llm_message llm_message_parent_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.llm_message
    ADD CONSTRAINT llm_message_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES transactions.llm_message(id);


--
-- Name: llm_message llm_message_session_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.llm_message
    ADD CONSTRAINT llm_message_session_id_fkey FOREIGN KEY (session_id) REFERENCES transactions.llm_session(id);


--
-- Name: llm_message llm_message_user_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.llm_message
    ADD CONSTRAINT llm_message_user_id_fkey FOREIGN KEY (user_id) REFERENCES transactions.users(id);


--
-- Name: llm_session llm_session_user_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.llm_session
    ADD CONSTRAINT llm_session_user_id_fkey FOREIGN KEY (user_id) REFERENCES transactions.users(id);


--
-- Name: transaction transaction_author_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_author_id_fkey FOREIGN KEY (author_id) REFERENCES transactions.users(id);


--
-- Name: transaction transaction_category_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_category_id_fkey FOREIGN KEY (category_id) REFERENCES transactions.category(id);


--
-- Name: transaction transaction_deleted_by_user_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_deleted_by_user_id_fkey FOREIGN KEY (deleted_by_user_id) REFERENCES transactions.users(id);


--
-- Name: transaction transaction_household_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_household_id_fkey FOREIGN KEY (household_id) REFERENCES transactions.household(id);


--
-- PostgreSQL database dump complete
--

\unrestrict dbmate


--
-- Dbmate schema migrations
--

INSERT INTO transactions.schema_migrations (version) VALUES
    ('20251130194702'),
    ('20260120030638'),
    ('20260505000000'),
    ('20260508000000'),
    ('20260510000000');
