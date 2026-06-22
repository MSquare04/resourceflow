DROP TRIGGER IF EXISTS trg_resource_unavailability_active_booking_check ON app.resource_unavailability;
DROP FUNCTION IF EXISTS app.prevent_active_booking_overlap_with_unavailability();

DROP TRIGGER IF EXISTS trg_resource_unavailability_resource_check ON app.resource_unavailability;
DROP FUNCTION IF EXISTS app.ensure_resource_is_bookable_for_unavailability();

DROP TABLE IF EXISTS app.resource_unavailability;

ALTER TABLE app.booking_rules
  DROP CONSTRAINT IF EXISTS chk_booking_rules_workday_window,
  DROP COLUMN IF EXISTS unrestricted_time,
  DROP COLUMN IF EXISTS workday_end,
  DROP COLUMN IF EXISTS workday_start;
