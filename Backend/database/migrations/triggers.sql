DELIMITER //

-- Drop existing triggers
DROP TRIGGER IF EXISTS after_table_insert//
DROP TRIGGER IF EXISTS after_table_update//
DROP TRIGGER IF EXISTS after_table_delete//
DROP TRIGGER IF EXISTS after_order_insert//
DROP TRIGGER IF EXISTS after_order_update//
DROP TRIGGER IF EXISTS after_order_delete//
DROP TRIGGER IF EXISTS after_payment_insert//
DROP TRIGGER IF EXISTS after_payment_update//
DROP TRIGGER IF EXISTS after_payment_delete//
DROP TRIGGER IF EXISTS after_receipt_insert//
DROP TRIGGER IF EXISTS after_receipt_update//
DROP TRIGGER IF EXISTS after_receipt_delete//

-- Create table trigger
CREATE TRIGGER after_table_insert 
AFTER INSERT ON tables
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('tables', NEW.id, 'INSERT');
END//

CREATE TRIGGER after_table_update
AFTER UPDATE ON tables
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('tables', NEW.id, 'UPDATE');
END//

CREATE TRIGGER after_table_delete
AFTER DELETE ON tables
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('tables', OLD.id, 'DELETE');
END//

CREATE TRIGGER after_order_insert 
AFTER INSERT ON orders
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('orders', NEW.id, 'INSERT');
END//

CREATE TRIGGER after_order_update
AFTER UPDATE ON orders
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('orders', NEW.id, 'UPDATE');
END//

CREATE TRIGGER after_order_delete
AFTER DELETE ON orders
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('orders', OLD.id, 'DELETE');
END//

CREATE TRIGGER after_payment_insert 
AFTER INSERT ON payments
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('payments', NEW.id, 'INSERT');
END//

CREATE TRIGGER after_payment_update
AFTER UPDATE ON payments
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('payments', NEW.id, 'UPDATE');
END//

CREATE TRIGGER after_payment_delete
AFTER DELETE ON payments
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('payments', OLD.id, 'DELETE');
END//

CREATE TRIGGER after_receipt_insert 
AFTER INSERT ON receipts
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('receipts', NEW.id, 'INSERT');
END//

CREATE TRIGGER after_receipt_update
AFTER UPDATE ON receipts
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('receipts', NEW.id, 'UPDATE');
END//

CREATE TRIGGER after_receipt_delete
AFTER DELETE ON receipts
FOR EACH ROW 
BEGIN
    INSERT INTO db_changes (table_name, record_id, action_type)
    VALUES ('receipts', OLD.id, 'DELETE');
END//

DELIMITER ; 