INSERT INTO app.resource_categories (code, name, description)
VALUES
  ('room', 'Помещения', ''),
  ('workspace', 'Рабочие зоны', ''),
  ('equipment', 'Оборудование', '')
ON CONFLICT (code) DO NOTHING;

INSERT INTO app.resource_types (category_id, code, name, description)
SELECT c.id, v.code, v.name, ''
FROM (
  VALUES
    ('room', 'meeting_room', 'Переговорная'),
    ('room', 'conference_hall', 'Конференц-зал'),
    ('workspace', 'desk', 'Рабочее место'),
    ('equipment', 'projector', 'Проектор'),
    ('equipment', 'laptop', 'Ноутбук')
) AS v(category_code, code, name)
INNER JOIN app.resource_categories c ON c.code = v.category_code
ON CONFLICT (code) DO NOTHING;
