ALTER TABLE app.booking_rules
  ADD COLUMN workday_start TIME NOT NULL DEFAULT TIME '07:00',
  ADD COLUMN workday_end TIME NOT NULL DEFAULT TIME '22:00',
  ADD COLUMN unrestricted_time BOOLEAN NOT NULL DEFAULT FALSE,
  ADD CONSTRAINT chk_booking_rules_workday_window
    CHECK (workday_start < workday_end);

CREATE TABLE app.resource_unavailability (
  id BIGSERIAL PRIMARY KEY,
  resource_id BIGINT NOT NULL,
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  reason TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_resource_unavailability_resource
    FOREIGN KEY (resource_id)
    REFERENCES app.resources (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT chk_resource_unavailability_time_range
    CHECK (start_at < end_at)
);

CREATE INDEX idx_resource_unavailability_resource_id
  ON app.resource_unavailability (resource_id);

CREATE INDEX idx_resource_unavailability_resource_start_at
  ON app.resource_unavailability (resource_id, start_at);

CREATE OR REPLACE FUNCTION app.ensure_resource_is_bookable_for_unavailability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  v_is_active BOOLEAN;
  v_is_bookable BOOLEAN;
BEGIN
  SELECT r.is_active, r.is_bookable
  INTO v_is_active, v_is_bookable
  FROM app.resources r
  WHERE r.id = NEW.resource_id;

  IF NOT FOUND THEN
    RETURN NEW;
  END IF;

  IF v_is_active IS NOT TRUE OR v_is_bookable IS NOT TRUE THEN
    RAISE EXCEPTION 'resource % is not active or not bookable', NEW.resource_id
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_resource_unavailability_resource_check
BEFORE INSERT OR UPDATE ON app.resource_unavailability
FOR EACH ROW
EXECUTE FUNCTION app.ensure_resource_is_bookable_for_unavailability();

CREATE OR REPLACE FUNCTION app.prevent_active_booking_overlap_with_unavailability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM app.bookings b
    WHERE b.resource_id = NEW.resource_id
      AND b.status IN ('pending', 'confirmed')
      AND b.start_at < NEW.end_at
      AND b.end_at > NEW.start_at
  ) THEN
    RAISE EXCEPTION 'resource unavailability conflicts with existing active bookings'
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_resource_unavailability_active_booking_check
BEFORE INSERT OR UPDATE ON app.resource_unavailability
FOR EACH ROW
EXECUTE FUNCTION app.prevent_active_booking_overlap_with_unavailability();
