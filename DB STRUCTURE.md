household_transaction // transacciones de la casa

- id --- int
- paid_by --- int --> user
- amount --- float
- category --- string
- is_paid --- bool
- amount_owed -- float
- budget_category_id -- fk --> budget_category
- household_id -- fk --> household

personal_transaction // transacciones personales

- id
- user_id
- category

user -- robert, val, etc // usuarios en general

- id
- discord_id
- name

household

- id -- pk
- name -- string

household_user // agrupacion de usuarios en una casa

<!-- - id -- pk -->

- household_id -- string, --> puede ser extraido a una tabla separada
- user_id -- fk --> user

budget

- id -- pk
- user -- fk --> user
- household_id -- fk --> household
- type -- string --> personal/household

budget_category

- id -- pk
- budget -- fk --> budget
- category_name -- strng
- allocation -- float

https://entgo.io/
