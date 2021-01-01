CREATE TABLE IF NOT EXISTS listings(
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    external_id INT NOT NULL,
    area_id INT NOT NULL REFERENCES areas(id),
    price INT NOT NULL,
    size DOUBLE PRECISION NOT NULL,
    rooms INT NOT NULL,
    visits INT NOT NULL,
    floor INT NOT NULL,
    listing_data JSONB,
    listing_details JSONB,
    date_accessed DATE NOT NULL DEFAULT CURRENT_DATE
)

