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
    type character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: budget_category; Type: TABLE; Schema: transactions; Owner: -
--

CREATE TABLE transactions.budget_category (
    id bigint NOT NULL,
    budget_id bigint,
    category_name character varying NOT NULL,
    allocation real NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: budget_category_id_seq; Type: SEQUENCE; Schema: transactions; Owner: -
--

ALTER TABLE transactions.budget_category ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME transactions.budget_category_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


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
    budget_category_id bigint,
    description character varying,
    transaction_date timestamp with time zone NOT NULL,
    transaction_id character varying NOT NULL,
    household_id bigint,
    notes text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


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
    discord_id character varying NOT NULL,
    name character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


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
-- Name: budget_category budget_category_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_category
    ADD CONSTRAINT budget_category_pkey PRIMARY KEY (id);


--
-- Name: budget budget_pkey; Type: CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget
    ADD CONSTRAINT budget_pkey PRIMARY KEY (id);


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
-- Name: idx_transaction_budget_category_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_transaction_budget_category_id ON transactions.transaction USING btree (budget_category_id);


--
-- Name: idx_transaction_household_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_transaction_household_id ON transactions.transaction USING btree (household_id);


--
-- Name: idx_users_discord_id; Type: INDEX; Schema: transactions; Owner: -
--

CREATE INDEX idx_users_discord_id ON transactions.users USING btree (discord_id);


--
-- Name: budget_category budget_category_budget_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget_category
    ADD CONSTRAINT budget_category_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES transactions.budget(id);


--
-- Name: budget budget_household_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.budget
    ADD CONSTRAINT budget_household_id_fkey FOREIGN KEY (household_id) REFERENCES transactions.household(id);


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
-- Name: transaction transaction_budget_category_id_fkey; Type: FK CONSTRAINT; Schema: transactions; Owner: -
--

ALTER TABLE ONLY transactions.transaction
    ADD CONSTRAINT transaction_budget_category_id_fkey FOREIGN KEY (budget_category_id) REFERENCES transactions.budget_category(id);


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
    ('20251130194702');
