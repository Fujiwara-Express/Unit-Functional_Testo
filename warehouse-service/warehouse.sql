CREATE TABLE items (
    item_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    location VARCHAR(50)
);

-- Masukkan data simulasi awal
INSERT INTO items (item_id, name, quantity, location) 
VALUES ('BRG-001', 'Laptop Gaming', 50, 'Rak-A1');