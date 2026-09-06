-- Hapus constraint UNIQUE bawaan kolom, ganti dengan index case-insensitive.
ALTER TABLE menu_items DROP CONSTRAINT IF EXISTS menu_items_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_menu_items_name_lower
    ON menu_items (LOWER(name));