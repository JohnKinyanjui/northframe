-- name: GetCommerceStats :one
select
    (select count(1) from orders where status = 'processing') as processing_orders,
    (select count(1) from products where stock <= reorder_point) as low_stock,
    (select coalesce(sum(total_cents), 0)::bigint from orders) as gross_cents,
    (select count(1) from orders) as total_orders;

-- name: ListOrders :many
select id, reference, customer_name, total_cents, status, destination,
       to_char(placed_at, 'DD Mon, HH24:MI') as placed_at
from orders
order by id desc
limit 12;

-- name: ListInventory :many
select id, sku, name, category, stock, reorder_point, price_cents,
       case when stock = 0 then 'out' when stock <= reorder_point then 'low' else 'healthy' end as health
from products
order by stock asc, name asc;

-- name: CreateProduct :one
insert into products (sku, name, category, stock, reorder_point, price_cents)
values (
    sqlc.arg(sku),
    sqlc.arg(name),
    sqlc.arg(category),
    sqlc.arg(stock),
    sqlc.arg(reorder_point),
    sqlc.arg(price_cents)
)
returning id, sku, name, category, stock, reorder_point, price_cents;
