-- DeliveryDesk Database Initialization
-- This script is executed by MySQL on first startup

CREATE DATABASE IF NOT EXISTS `deliverydesk` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- If database already exists with wrong collation, fix it
ALTER DATABASE `deliverydesk` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- Grant all privileges to the application user
GRANT ALL PRIVILEGES ON `deliverydesk`.* TO 'deliverydesk'@'%';
FLUSH PRIVILEGES;
