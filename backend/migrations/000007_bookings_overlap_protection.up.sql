CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE app.bookings
  ADD CONSTRAINT bookings_no_overlaps_active
  EXCLUDE USING gist (
    resource_id WITH =,
    tstzrange(start_at, end_at, '[)') WITH &&
  )
  WHERE (status IN ('pending', 'confirmed'));
