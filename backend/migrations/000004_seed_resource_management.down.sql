DELETE FROM app.resource_types
WHERE code IN ('meeting_room', 'conference_hall', 'desk', 'projector', 'laptop');

DELETE FROM app.resource_categories
WHERE code IN ('room', 'workspace', 'equipment');
