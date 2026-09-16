ALTER TABLE bookings
ADD COLUMN user_name VARCHAR(255),
ADD COLUMN phone VARCHAR(50);

CREATE INDEX idx_bookings_user_status ON bookings(user_id, status, created_at DESC);
CREATE INDEX idx_bookings_availability ON bookings(zone, table_name, time_slot, status);
