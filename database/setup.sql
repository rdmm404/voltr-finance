SET search_path TO transactions, public;

WITH inserted_household AS (
    INSERT INTO household (name) VALUES ('Voltr')
    RETURNING id
),
inserted_users AS (
    INSERT INTO users
    (discord_id, name)
    VALUES
    ('263106741711929351', 'Robert'),
    ('562395673614352396', 'Val'),
    -- testers (solo los reales)!!!!
    ('488550621113221130', 'SlowJeans92'),
    ('253599462780436482', 'Levonor'),
    ('362037506276851713', 'Lucas'),
    ('602335495220887552', 'Rami'),
    ('162771549038968832', 'Keitx')

    RETURNING id
)

INSERT INTO household_user
(user_id, household_id)
SELECT iu.id, h.id
FROM inserted_users iu, inserted_household h;
