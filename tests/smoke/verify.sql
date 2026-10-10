DO $$
BEGIN
  IF current_database() <> 'j_smoke' THEN
    RAISE EXCEPTION 'Use the dedicated j_smoke database';
  END IF;
  IF (SELECT count(*) FROM stocks) <> 3
     OR NOT EXISTS (SELECT 1 FROM stocks WHERE product_id=1 AND quantity=7)
     OR NOT EXISTS (SELECT 1 FROM stocks WHERE product_id=2 AND quantity=10)
     OR NOT EXISTS (SELECT 1 FROM stocks WHERE product_id=3 AND quantity=0) THEN
    RAISE EXCEPTION 'Final stock does not match successful decrements';
  END IF;
END $$;
SELECT product_id,quantity FROM stocks ORDER BY product_id;
