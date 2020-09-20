CREATE TABLE IF NOT EXISTS listings(
    id SERIAL PRIMARY KEY,
    area_id INT NOT NULL,
    price INT NOT NULL,
    listing_data JSONB,
    listing_details JSONB,
    date_accessed DATE NOT NULL DEFAULT CURRENT_DATE,
    CONSTRAINT fk_area FOREIGN KEY(area_id) REFERENCES areas(area_id)
)
