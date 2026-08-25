insert into products (sku, name, category, stock, reorder_point, price_cents) values
('ST-044', 'Stoneware Carafe', 'Table', 0, 12, 8900),
('LN-112', 'Washed Linen Throw', 'Textiles', 4, 10, 12400),
('LT-018', 'Travertine Table Lamp', 'Lighting', 7, 8, 21900),
('WD-071', 'Oak Valet Tray', 'Objects', 22, 6, 6800),
('GL-205', 'Ripple Glass Set', 'Table', 38, 12, 7600);
-- northframe:split
insert into orders (reference, customer_name, total_cents, status, destination, placed_at) values
('RLY-10482', 'Mara Studio', 24800, 'processing', 'Berlin, DE', now() - interval '18 minutes'),
('RLY-10481', 'Field Notes Co.', 12950, 'processing', 'Austin, US', now() - interval '42 minutes'),
('RLY-10480', 'Common Ground', 8600, 'processing', 'Nairobi, KE', now() - interval '1 hour'),
('RLY-10479', 'Atelier Noma', 41200, 'dispatched', 'Paris, FR', now() - interval '2 hours'),
('RLY-10478', 'Onda Market', 19750, 'ready', 'Lisbon, PT', now() - interval '3 hours');
