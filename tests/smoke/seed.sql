BEGIN;
DO $$
BEGIN
  IF current_database() <> 'j_smoke' THEN
    RAISE EXCEPTION 'Use the dedicated j_smoke database';
  END IF;
  IF EXISTS (SELECT 1 FROM categories) OR EXISTS (SELECT 1 FROM products) OR EXISTS (SELECT 1 FROM stocks) THEN
    RAISE EXCEPTION 'Smoke fixtures require an empty database';
  END IF;
END $$;
INSERT INTO categories(id,name) OVERRIDING SYSTEM VALUE VALUES (1,'Books'),(2,'Food');
INSERT INTO products(id,name,price,category_id) OVERRIDING SYSTEM VALUE VALUES
  (1,'Go入門',200,1),(2,'Go実践',100,1),(3,'Rice',300,2);
INSERT INTO stocks(product_id,quantity) VALUES (1,10),(2,10),(3,0);
SELECT setval(pg_get_serial_sequence('categories','id'),2);
SELECT setval(pg_get_serial_sequence('products','id'),3);
ANALYZE;
COMMIT;
