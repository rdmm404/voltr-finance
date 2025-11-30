SET search_path TO transactions, public;

WITH voltr_household_id AS (
    INSERT INTO household (name, guild_id) VALUES ('Voltr', '721764134357827696') RETURNING id
),
groster_household_id AS (
    INSERT INTO household (name, guild_id) VALUES ('Groster', '1402113769160572958') RETURNING id
),
inserted_users AS (
    INSERT INTO users
    (discord_id, name)
    VALUES
    ('263106741711929351', 'Robert'),
    ('562395673614352396', 'Val')
    RETURNING id
),
inserted_users_testers AS (
    INSERT INTO users
    (discord_id, name)
    VALUES
    -- testers (solo los reales)!!!!
    ('488550621113221130', 'SlowJeans92'),
    ('253599462780436482', 'Levonor'),
    ('362037506276851713', 'Lucas'),
    ('602335495220887552', 'Rami'),
    ('162771549038968832', 'Keitx')
    RETURNING id
)
INSERT INTO household_user (user_id, household_id)
SELECT iu.id, vh.id FROM inserted_users iu, voltr_household_id vh
UNION ALL
SELECT iu.id, gh.id FROM inserted_users iu, groster_household_id gh
UNION ALL
SELECT iut.id, gh.id FROM inserted_users_testers iut, groster_household_id gh;