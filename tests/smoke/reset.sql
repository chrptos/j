BEGIN;
DO $$
BEGIN
  IF current_database() <> 'j_smoke' THEN
    RAISE EXCEPTION 'Use the dedicated j_smoke database';
  END IF;
  IF (SELECT count(*) FROM products) <> 3 OR (SELECT count(*) FROM stocks) <> 3
     OR NOT EXISTS (SELECT 1 FROM products WHERE id=1 AND name='Go入門')
     OR NOT EXISTS (SELECT 1 FROM products WHERE id=2 AND name='Go実践')
     OR NOT EXISTS (SELECT 1 FROM products WHERE id=3 AND name='Rice') THEN
    RAISE EXCEPTION 'Smoke fixtures are missing or changed';
  END IF;
END $$;
UPDATE stocks SET quantity=CASE WHEN product_id=3 THEN 0 ELSE 10 END;
COMMIT;
