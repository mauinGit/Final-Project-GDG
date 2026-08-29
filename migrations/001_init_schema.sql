CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_users_role CHECK (role IN ('kasir', 'koki'))
);

CREATE TABLE IF NOT EXISTS orders (
    id            BIGSERIAL PRIMARY KEY,
    queue_number  INT NOT NULL,
    customer_name VARCHAR(100) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_orders_status CHECK (status IN ('pending', 'cooking', 'done', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at);

CREATE TABLE IF NOT EXISTS order_items (
    id        BIGSERIAL PRIMARY KEY,
    order_id  BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_name VARCHAR(100) NOT NULL,
    quantity  INT NOT NULL,
    note      VARCHAR(255),
    CONSTRAINT chk_order_items_quantity CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);