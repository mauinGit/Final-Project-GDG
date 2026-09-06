ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS subtotal       INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount       INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total          INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20) NOT NULL DEFAULT 'cash',
    ADD COLUMN IF NOT EXISTS amount_paid    INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS change_amount  INT NOT NULL DEFAULT 0;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS chk_orders_payment_method;

ALTER TABLE orders
    ADD CONSTRAINT chk_orders_payment_method
    CHECK (payment_method IN ('cash', 'non_cash'));

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS chk_orders_money;

ALTER TABLE orders
    ADD CONSTRAINT chk_orders_money
    CHECK (
        subtotal >= 0
        AND discount >= 0
        AND discount <= subtotal
        AND total = subtotal - discount
    );