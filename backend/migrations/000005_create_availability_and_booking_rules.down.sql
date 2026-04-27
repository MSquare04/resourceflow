DROP TABLE IF EXISTS app.booking_rules;

DROP TRIGGER IF EXISTS trg_resource_availability_resource_check ON app.resource_availability;
DROP FUNCTION IF EXISTS app.ensure_resource_is_bookable_for_availability();
DROP TABLE IF EXISTS app.resource_availability;
