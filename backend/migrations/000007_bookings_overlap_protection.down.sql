ALTER TABLE app.bookings
  DROP CONSTRAINT IF EXISTS bookings_no_overlaps_active;
