CREATE TABLE app.bookings (
  id BIGSERIAL PRIMARY KEY,
  resource_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  purpose TEXT NULL,
  status VARCHAR(20) NOT NULL,
  approved_by_user_id BIGINT NULL,
  approved_at TIMESTAMPTZ NULL,
  cancelled_at TIMESTAMPTZ NULL,
  completed_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_bookings_resource
    FOREIGN KEY (resource_id)
    REFERENCES app.resources (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT fk_bookings_user
    FOREIGN KEY (user_id)
    REFERENCES app.users (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT fk_bookings_approved_by_user
    FOREIGN KEY (approved_by_user_id)
    REFERENCES app.users (id)
    ON UPDATE CASCADE
    ON DELETE SET NULL,
  CONSTRAINT chk_bookings_time_range
    CHECK (start_at < end_at),
  CONSTRAINT chk_bookings_status
    CHECK (status IN ('pending', 'confirmed', 'rejected', 'cancelled', 'completed'))
);

CREATE INDEX idx_bookings_resource_time
  ON app.bookings (resource_id, start_at, end_at);

CREATE INDEX idx_bookings_user_id
  ON app.bookings (user_id);

CREATE INDEX idx_bookings_user_status
  ON app.bookings (user_id, status);

CREATE INDEX idx_bookings_resource_status
  ON app.bookings (resource_id, status);
