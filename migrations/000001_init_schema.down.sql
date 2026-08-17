-- 000001_init_schema.down.sql
-- Drop in reverse dependency order (staff and patients reference hospitals).
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS hospitals;
