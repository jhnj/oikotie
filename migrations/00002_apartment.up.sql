CREATE TABLE IF NOT EXISTS listings(
    id SERIAL PRIMARY KEY,
    area_id INT NOT NULL REFERENCES areas(id),
    price INT NOT NULL,
    listing_data JSONB,
    listing_details JSONB,
    date_accessed DATE NOT NULL DEFAULT CURRENT_DATE
)
