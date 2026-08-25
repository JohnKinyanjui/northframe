create table products (
    id bigserial primary key,
    sku text not null unique,
    name text not null,
    category text not null,
    stock integer not null,
    reorder_point integer not null,
    price_cents bigint not null
);
-- northframe:split

create table orders (
    id bigserial primary key,
    reference text not null unique,
    customer_name text not null,
    total_cents bigint not null,
    status text not null,
    destination text not null,
    placed_at timestamptz not null default now()
);
