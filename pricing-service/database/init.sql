-- Buat tabel untuk Zona
CREATE TABLE zones (
    id SERIAL PRIMARY KEY,
    origin VARCHAR(10),
    destination VARCHAR(10),
    zone_id VARCHAR(10) NOT NULL
);

-- Buat tabel untuk Tarif
CREATE TABLE rates (
    id SERIAL PRIMARY KEY,
    zone_id VARCHAR(10),
    service_type VARCHAR(20),
    price_per_kg DECIMAL,
    min_weight DECIMAL,
    max_weight DECIMAL,
    max_length DECIMAL,
    max_width DECIMAL,
    max_height DECIMAL
);

-- Buat tabel untuk Denda (Surcharge)
CREATE TABLE surcharges (
    id SERIAL PRIMARY KEY,
    surcharge_type VARCHAR(20),
    surcharge_value DECIMAL
);

-- ISI DATA CONTOH (Agar bisa dites)
INSERT INTO zones (origin, destination, zone_id) VALUES ('CGK', 'BDO', 'Z1');
INSERT INTO rates (zone_id, service_type, price_per_kg, min_weight, max_weight, max_length, max_width, max_height) 
VALUES ('Z1', 'REG', 10000, 1, 50, 100, 100, 100);
INSERT INTO surcharges (surcharge_type, surcharge_value) VALUES ('OVERSIZE', 50000);