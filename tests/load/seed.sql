BEGIN;
SELECT set_config('j.products', :'products', true),set_config('j.seed', :'seed', true),set_config('j.distribution', :'distribution', true),set_config('j.stock', :'stock', true);
DO $$
BEGIN
  IF current_database() <> 'j_load' THEN RAISE EXCEPTION 'Use the dedicated j_load database'; END IF;
  IF current_setting('j.products')::bigint NOT BETWEEN 1 AND 1000000
     OR current_setting('j.seed')::bigint NOT BETWEEN 0 AND 2147483647
     OR current_setting('j.stock')::bigint NOT BETWEEN 1 AND 2147483647
     OR current_setting('j.distribution') NOT IN ('uniform','hot') THEN
    RAISE EXCEPTION 'Invalid seed parameters';
  END IF;
  IF EXISTS(SELECT 1 FROM categories) OR EXISTS(SELECT 1 FROM products) OR EXISTS(SELECT 1 FROM stocks) THEN
    RAISE EXCEPTION 'Seed requires an empty database';
  END IF;
END $$;
INSERT INTO categories(id,name) OVERRIDING SYSTEM VALUE SELECT i,'カテゴリ-'||i FROM generate_series(1,100) AS i;
INSERT INTO products(id,name,price,category_id) OVERRIDING SYSTEM VALUE
SELECT i,
       '商品-'||lpad(i::text,7,'0')||'-グループ-'||lpad((('x'||substr(md5('name:'||:'seed'||':'||i),1,8))::bit(32)::bigint%100)::text,2,'0'),
       (('x'||substr(md5('price:'||:'seed'||':'||i),1,8))::bit(32)::bigint%100001)::integer,
       CASE WHEN :'distribution'='hot' THEN
         CASE WHEN (i-1)%5<>4 THEN 1 ELSE 2+((i-1)/5+:'seed'::bigint)%99 END
         ELSE 1+(i-1+:'seed'::bigint)%100 END
FROM generate_series(1,:'products'::bigint) AS i;
INSERT INTO stocks(product_id,quantity) SELECT id,:'stock'::integer FROM products;
SELECT setval(pg_get_serial_sequence('categories','id'),100);
SELECT setval(pg_get_serial_sequence('products','id'),:'products'::bigint);
COMMIT;
ANALYZE;
-- 記録用：生成条件・カテゴリ分布・価格範囲・部分一致の実際のヒット数
SELECT :'products' AS products,:'seed' AS seed,:'distribution' AS distribution,:'stock' AS initial_stock;
SELECT category_id,count(*) FROM products GROUP BY category_id ORDER BY category_id;
SELECT min(price),max(price),count(*) FILTER(WHERE name LIKE '%グループ-00%') AS keyword_hits FROM products;
