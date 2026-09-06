CREATE TABLE IF NOT EXISTS menu_items (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100) UNIQUE NOT NULL,
    price      INT NOT NULL,
    category   VARCHAR(50) NOT NULL,
    image_url  VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_menu_items_price CHECK (price >= 0)
);

CREATE INDEX IF NOT EXISTS idx_menu_items_category ON menu_items (category);

-- order_items sekarang merujuk ke menu, dan menyimpan harga saat transaksi.
ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS menu_item_id BIGINT REFERENCES menu_items(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS price_at_order INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_order_items_menu_item_id ON order_items (menu_item_id);