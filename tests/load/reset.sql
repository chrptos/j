BEGIN;
SELECT set_config('j.stock', :'stock', true);
DO $$
BEGIN
  IF current_database()<>'j_load' THEN RAISE EXCEPTION 'Use the dedicated j_load database'; END IF;
  IF current_setting('j.stock')::bigint NOT BETWEEN 1 AND 2147483647 THEN RAISE EXCEPTION 'Invalid stock'; END IF;
  IF NOT EXISTS(SELECT 1 FROM products) OR EXISTS(SELECT 1 FROM products p LEFT JOIN stocks s ON s.product_id=p.id WHERE s.product_id IS NULL) THEN
    RAISE EXCEPTION 'Products or stocks are missing';
  END IF;
END $$;
UPDATE stocks SET quantity=:'stock'::integer;
COMMIT;
