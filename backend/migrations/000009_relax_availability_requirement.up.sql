CREATE OR REPLACE FUNCTION app.ensure_active_bookings_stay_covered_on_availability_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  v_resource_id BIGINT;
BEGIN
  v_resource_id := COALESCE(NEW.resource_id, OLD.resource_id);

  IF NOT EXISTS (
    SELECT 1
    FROM app.resource_availability ra_remaining
    WHERE ra_remaining.resource_id = v_resource_id
      AND (TG_OP <> 'DELETE' OR ra_remaining.id <> OLD.id)
  ) THEN
    IF TG_OP = 'DELETE' THEN
      RETURN OLD;
    END IF;

    RETURN NEW;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM app.bookings b
    WHERE b.resource_id = v_resource_id
      AND b.status IN ('pending', 'confirmed')
      AND NOT (
        (TG_OP = 'UPDATE' AND NEW.start_at <= b.start_at AND NEW.end_at >= b.end_at)
        OR EXISTS (
          SELECT 1
          FROM app.resource_availability ra
          WHERE ra.resource_id = v_resource_id
            AND ra.id <> OLD.id
            AND ra.start_at <= b.start_at
            AND ra.end_at >= b.end_at
        )
      )
  ) THEN
    RAISE EXCEPTION 'availability change would leave active bookings outside availability'
      USING ERRCODE = '23514';
  END IF;

  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;

  RETURN NEW;
END;
$$;
