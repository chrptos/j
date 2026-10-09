BEGIN;

CREATE TABLE categories (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    name varchar(100) NOT NULL CHECK (name <> '')
);

CREATE TABLE products (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    name varchar(200) NOT NULL CHECK (name <> ''),
    price integer NOT NULL CHECK (price >= 0),
    category_id bigint NOT NULL REFERENCES categories (id)
);

CREATE TABLE stocks (
    product_id bigint PRIMARY KEY REFERENCES products (id),
    quantity integer NOT NULL CHECK (quantity >= 0)
);

COMMIT;
