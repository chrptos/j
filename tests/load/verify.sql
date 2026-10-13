SELECT set_config('j.stock', :'stock', false),set_config('j.successes', :'successes', false),set_config('j.products', :'products', false);
DO $$
DECLARE n bigint; remaining bigint;
BEGIN
  IF current_database()<>'j_load' THEN RAISE EXCEPTION 'Use the dedicated j_load database'; END IF;
  IF current_setting('j.stock')::bigint NOT BETWEEN 1 AND 2147483647 OR current_setting('j.successes')::bigint<0 OR current_setting('j.products')::bigint NOT BETWEEN 1 AND 1000000 THEN RAISE EXCEPTION 'Invalid verification parameters'; END IF;
  SELECT count(*),sum(quantity) INTO n,remaining FROM stocks;
  IF n<>current_setting('j.products')::bigint OR n<>(SELECT count(*) FROM products)
     OR EXISTS(SELECT 1 FROM stocks WHERE quantity<0 OR quantity>current_setting('j.stock')::bigint)
     OR remaining<>n*current_setting('j.stock')::bigint-current_setting('j.successes')::bigint THEN
    RAISE EXCEPTION 'Stock does not match successful decrements';
  END IF;
END $$;
SELECT count(*) AS products,sum(quantity) AS remaining_stock,min(quantity) AS minimum_stock FROM stocks;
