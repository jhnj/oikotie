CREATE TABLE IF NOT EXISTS areas(
    id serial PRIMARY KEY,
    external_id int NOT NULL,
    name text NOT NULL,
    city text NOT NULL,
    card_type int NOT NULL
);
