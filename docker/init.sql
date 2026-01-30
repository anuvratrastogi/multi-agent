-- Purchase Orders Sample Schema and Data
-- This script initializes the database with tables and sample data for testing

-- Create ENUM types
CREATE TYPE order_status AS ENUM ('pending', 'approved', 'shipped', 'delivered', 'cancelled');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'refunded', 'failed');

-- Suppliers table
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    unit_price DECIMAL(10, 2) NOT NULL,
    supplier_id INTEGER REFERENCES suppliers(id),
    stock_quantity INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Customers table
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(50),
    company VARCHAR(255),
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Purchase Orders table
CREATE TABLE purchase_orders (
    id SERIAL PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id INTEGER REFERENCES customers(id),
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expected_delivery DATE,
    status order_status DEFAULT 'pending',
    payment_status payment_status DEFAULT 'pending',
    subtotal DECIMAL(12, 2),
    tax DECIMAL(12, 2),
    shipping_cost DECIMAL(10, 2) DEFAULT 0,
    total_amount DECIMAL(12, 2),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Order Items table
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    discount_percent DECIMAL(5, 2) DEFAULT 0,
    line_total DECIMAL(12, 2) NOT NULL
);

-- Order History table for tracking status changes
CREATE TABLE order_history (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES purchase_orders(id) ON DELETE CASCADE,
    old_status order_status,
    new_status order_status,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    changed_by VARCHAR(100)
);

-- =============================================================================
-- INSERT SAMPLE DATA
-- =============================================================================

-- Insert Suppliers
INSERT INTO suppliers (name, contact_email, contact_phone, address, city, country) VALUES
('TechParts Global', 'sales@techparts.com', '+1-555-0101', '123 Industry Ave', 'San Francisco', 'USA'),
('ElectroSupply Co', 'orders@electrosupply.com', '+1-555-0102', '456 Electronics Blvd', 'Seattle', 'USA'),
('Hardware Direct', 'contact@hardwaredirect.com', '+44-20-1234567', '789 Tech Park', 'London', 'UK'),
('Asian Components Ltd', 'info@asiancomponents.com', '+86-21-5555678', '321 Manufacturing Zone', 'Shanghai', 'China'),
('EuroTech Supplies', 'sales@eurotech.eu', '+49-30-987654', '654 Industrial Park', 'Berlin', 'Germany');

-- Insert Products
INSERT INTO products (sku, name, description, category, unit_price, supplier_id, stock_quantity) VALUES
('LAPTOP-001', 'Business Laptop Pro', '15.6" business laptop with 16GB RAM', 'Electronics', 1299.99, 1, 50),
('LAPTOP-002', 'Developer Workstation', '17" high-performance laptop', 'Electronics', 2499.99, 1, 25),
('MONITOR-001', '27" 4K Monitor', 'Ultra HD display with USB-C', 'Electronics', 549.99, 2, 100),
('MONITOR-002', '32" Curved Monitor', 'QHD curved gaming monitor', 'Electronics', 699.99, 2, 75),
('KEYBOARD-001', 'Mechanical Keyboard', 'RGB mechanical keyboard', 'Accessories', 149.99, 3, 200),
('MOUSE-001', 'Wireless Ergonomic Mouse', 'Bluetooth ergonomic mouse', 'Accessories', 79.99, 3, 300),
('HEADSET-001', 'Noise Cancelling Headset', 'Wireless ANC headphones', 'Audio', 299.99, 4, 150),
('WEBCAM-001', '4K Webcam Pro', 'Ultra HD webcam with mic', 'Electronics', 199.99, 4, 180),
('DOCK-001', 'USB-C Docking Station', '12-in-1 docking station', 'Accessories', 249.99, 5, 120),
('CABLE-001', 'USB-C Cable Pack', '3-pack USB-C cables', 'Accessories', 29.99, 5, 500);

-- Insert Customers
INSERT INTO customers (name, email, phone, company, address, city, country) VALUES
('John Smith', 'john.smith@acme.com', '+1-555-1001', 'Acme Corporation', '100 Main St', 'New York', 'USA'),
('Sarah Johnson', 'sarah.j@techstart.io', '+1-555-1002', 'TechStart Inc', '200 Innovation Way', 'Austin', 'USA'),
('Michael Chen', 'mchen@globaltech.com', '+1-555-1003', 'Global Tech Solutions', '300 Enterprise Blvd', 'San Jose', 'USA'),
('Emma Williams', 'emma@designstudio.co.uk', '+44-20-5551004', 'Design Studio Ltd', '50 Creative Lane', 'London', 'UK'),
('Hans Mueller', 'h.mueller@germantec.de', '+49-30-5551005', 'German Tech GmbH', '75 Technik Strasse', 'Munich', 'Germany'),
('Yuki Tanaka', 'ytanaka@tokyosoft.jp', '+81-3-5551006', 'Tokyo Software Co', '888 Shibuya Center', 'Tokyo', 'Japan'),
('Maria Garcia', 'mgarcia@spanishit.es', '+34-91-5551007', 'Spanish IT Solutions', '42 Calle Mayor', 'Madrid', 'Spain'),
('Alex Brown', 'alex.brown@startup.io', '+1-555-1008', 'Startup Ventures', '500 Venture Way', 'Boston', 'USA');

-- Insert Purchase Orders with varied dates for good chart data
INSERT INTO purchase_orders (order_number, customer_id, order_date, expected_delivery, status, payment_status, subtotal, tax, shipping_cost, total_amount, notes) VALUES
('PO-2024-001', 1, '2024-01-15', '2024-01-22', 'delivered', 'paid', 3899.97, 311.99, 50.00, 4261.96, 'Q1 equipment refresh'),
('PO-2024-002', 2, '2024-01-20', '2024-01-27', 'delivered', 'paid', 1549.98, 123.99, 25.00, 1698.97, 'New hire setup'),
('PO-2024-003', 3, '2024-02-01', '2024-02-08', 'delivered', 'paid', 5499.95, 439.99, 75.00, 6014.94, 'Office expansion'),
('PO-2024-004', 4, '2024-02-15', '2024-02-22', 'delivered', 'paid', 2299.97, 183.99, 100.00, 2583.96, 'Design team upgrade'),
('PO-2024-005', 5, '2024-03-01', '2024-03-08', 'delivered', 'paid', 4199.96, 335.99, 150.00, 4685.95, 'Engineering department'),
('PO-2024-006', 1, '2024-03-15', '2024-03-22', 'delivered', 'paid', 899.97, 71.99, 25.00, 996.96, 'Accessory restock'),
('PO-2024-007', 6, '2024-04-01', '2024-04-10', 'delivered', 'paid', 7499.95, 599.99, 200.00, 8299.94, 'Annual procurement'),
('PO-2024-008', 2, '2024-04-15', '2024-04-22', 'delivered', 'paid', 1099.98, 87.99, 30.00, 1217.97, 'Monitor upgrade'),
('PO-2024-009', 7, '2024-05-01', '2024-05-08', 'shipped', 'paid', 3299.97, 263.99, 120.00, 3683.96, 'Branch office setup'),
('PO-2024-010', 3, '2024-05-15', '2024-05-22', 'shipped', 'paid', 1799.97, 143.99, 40.00, 1983.96, 'Developer tools'),
('PO-2024-011', 8, '2024-06-01', '2024-06-10', 'approved', 'paid', 2599.98, 207.99, 50.00, 2857.97, 'Startup kit'),
('PO-2024-012', 4, '2024-06-15', '2024-06-25', 'approved', 'pending', 4999.95, 399.99, 100.00, 5499.94, 'Design workstations'),
('PO-2024-013', 5, '2024-07-01', '2024-07-12', 'pending', 'pending', 1299.99, 103.99, 25.00, 1428.98, 'Single laptop order'),
('PO-2024-014', 6, '2024-07-10', '2024-07-20', 'pending', 'pending', 599.98, 47.99, 20.00, 667.97, 'Peripheral bundle'),
('PO-2024-015', 1, '2024-07-15', '2024-07-25', 'cancelled', 'refunded', 2499.99, 199.99, 50.00, 2749.98, 'Order cancelled by customer');

-- Insert Order Items
INSERT INTO order_items (order_id, product_id, quantity, unit_price, discount_percent, line_total) VALUES
-- PO-2024-001
(1, 1, 3, 1299.99, 0, 3899.97),
-- PO-2024-002
(2, 3, 2, 549.99, 0, 1099.98),
(2, 5, 3, 149.99, 0, 449.97),
-- PO-2024-003
(3, 2, 2, 2499.99, 0, 4999.98),
(3, 9, 2, 249.99, 0, 499.98),
-- PO-2024-004
(4, 4, 2, 699.99, 0, 1399.98),
(4, 8, 3, 199.99, 0, 599.97),
(4, 6, 4, 79.99, 5, 303.96),
-- PO-2024-005
(5, 1, 2, 1299.99, 0, 2599.98),
(5, 3, 2, 549.99, 0, 1099.98),
(5, 9, 2, 249.99, 0, 499.98),
-- PO-2024-006
(6, 5, 3, 149.99, 0, 449.97),
(6, 6, 3, 79.99, 0, 239.97),
(6, 10, 7, 29.99, 0, 209.93),
-- PO-2024-007
(7, 2, 3, 2499.99, 0, 7499.97),
-- PO-2024-008
(8, 3, 2, 549.99, 0, 1099.98),
-- PO-2024-009
(9, 1, 1, 1299.99, 0, 1299.99),
(9, 3, 2, 549.99, 0, 1099.98),
(9, 7, 3, 299.99, 0, 899.97),
-- PO-2024-010
(10, 5, 5, 149.99, 0, 749.95),
(10, 6, 5, 79.99, 0, 399.95),
(10, 9, 2, 249.99, 5, 474.98),
(10, 10, 6, 29.99, 0, 179.94),
-- PO-2024-011
(11, 1, 2, 1299.99, 0, 2599.98),
-- PO-2024-012
(12, 2, 2, 2499.99, 0, 4999.98),
-- PO-2024-013
(13, 1, 1, 1299.99, 0, 1299.99),
-- PO-2024-014
(14, 5, 2, 149.99, 0, 299.98),
(14, 6, 2, 79.99, 0, 159.98),
(14, 10, 5, 29.99, 5, 142.45),
-- PO-2024-015
(15, 2, 1, 2499.99, 0, 2499.99);

-- Insert Order History
INSERT INTO order_history (order_id, old_status, new_status, changed_by) VALUES
(1, 'pending', 'approved', 'admin'),
(1, 'approved', 'shipped', 'warehouse'),
(1, 'shipped', 'delivered', 'logistics'),
(2, 'pending', 'approved', 'admin'),
(2, 'approved', 'shipped', 'warehouse'),
(2, 'shipped', 'delivered', 'logistics'),
(9, 'pending', 'approved', 'admin'),
(9, 'approved', 'shipped', 'warehouse'),
(15, 'pending', 'cancelled', 'customer');

-- Create useful views
CREATE VIEW monthly_sales AS
SELECT 
    DATE_TRUNC('month', order_date) AS month,
    COUNT(*) AS total_orders,
    SUM(total_amount) AS revenue,
    AVG(total_amount) AS avg_order_value
FROM purchase_orders
WHERE status != 'cancelled'
GROUP BY DATE_TRUNC('month', order_date)
ORDER BY month;

CREATE VIEW top_products AS
SELECT 
    p.name AS product_name,
    p.category,
    SUM(oi.quantity) AS total_sold,
    SUM(oi.line_total) AS total_revenue
FROM order_items oi
JOIN products p ON oi.product_id = p.id
JOIN purchase_orders po ON oi.order_id = po.id
WHERE po.status != 'cancelled'
GROUP BY p.id, p.name, p.category
ORDER BY total_revenue DESC;

CREATE VIEW customer_summary AS
SELECT 
    c.name AS customer_name,
    c.company,
    c.country,
    COUNT(po.id) AS total_orders,
    SUM(po.total_amount) AS total_spent
FROM customers c
LEFT JOIN purchase_orders po ON c.id = po.customer_id AND po.status != 'cancelled'
GROUP BY c.id, c.name, c.company, c.country
ORDER BY total_spent DESC NULLS LAST;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;
