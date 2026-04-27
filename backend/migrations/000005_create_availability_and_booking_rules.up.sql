CREATE TABLE app.resource_availability (
  id BIGSERIAL PRIMARY KEY,
  resource_id BIGINT NOT NULL,
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_resource_availability_resource
    FOREIGN KEY (resource_id)
    REFERENCES app.resources (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT chk_resource_availability_time_range
    CHECK (start_at < end_at)
);

CREATE INDEX idx_resource_availability_resource_id
  ON app.resource_availability (resource_id);

CREATE INDEX idx_resource_availability_resource_start_at
  ON app.resource_availability (resource_id, start_at);

CREATE OR REPLACE FUNCTION app.ensure_resource_is_bookable_for_availability()
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

CREATE TRIGGER trg_resource_availability_resource_check
BEFORE INSERT OR UPDATE ON app.resource_availability
FOR EACH ROW
EXECUTE FUNCTION app.ensure_resource_is_bookable_for_availability();

CREATE TABLE app.booking_rules (
  id BIGSERIAL PRIMARY KEY,
  resource_type_id BIGINT NOT NULL,
  min_duration_minutes INTEGER NOT NULL,
  max_duration_minutes INTEGER NOT NULL,
  max_active_bookings_per_user INTEGER NOT NULL,
  requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
  booking_horizon_days INTEGER NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_booking_rules_resource_type
    FOREIGN KEY (resource_type_id)
    REFERENCES app.resource_types (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT chk_booking_rules_min_duration
    CHECK (min_duration_minutes > 0),
  CONSTRAINT chk_booking_rules_max_duration
    CHECK (max_duration_minutes >= min_duration_minutes),
  CONSTRAINT chk_booking_rules_max_active_per_user
    CHECK (max_active_bookings_per_user >= 1),
  CONSTRAINT chk_booking_rules_booking_horizon_days
    CHECK (booking_horizon_days >= 0)
);

CREATE INDEX idx_booking_rules_resource_type_id
  ON app.booking_rules (resource_type_id);
