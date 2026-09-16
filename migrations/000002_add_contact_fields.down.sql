DROP INDEX IF EXISTS idx_bookings_availability;
DROP INDEX IF EXISTS idx_bookings_user_status;

ALTER TABLE bookings
DROP COLUMN IF EXISTS phone,
DROP COLUMN IF EXISTS user_name;
