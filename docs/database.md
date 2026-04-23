# Структура базы данных ResourceFlow

## Назначение документа

Документ описывает основные сущности базы данных, их назначение и ключевые связи.

---

## Общий подход

В качестве основной СУБД используется PostgreSQL.

Структура базы данных должна обеспечивать:
- хранение основных сущностей предметной области;
- поддержку связей между сущностями;
- соблюдение бизнес-правил;
- возможность дальнейшего расширения.

---

## Основные сущности

### 1. roles
Справочник ролей пользователей.

Основные поля:
- id
- code
- name
- description

---

### 2. departments
Справочник подразделений.

Основные поля:
- id
- name
- description
- is_active
- created_at
- updated_at

---

### 3. users
Пользователи системы.

Основные поля:
- id
- full_name
- email
- password_hash
- role_id
- department_id
- is_active
- created_at
- updated_at

Связи:
- пользователь принадлежит роли;
- пользователь может быть связан с подразделением.

---

### 4. resource_categories
Категории ресурсов.

Основные поля:
- id
- name
- description
- is_active
- created_at
- updated_at

---

### 5. booking_rules
Правила бронирования.

Основные поля:
- id
- name
- min_duration_minutes
- max_duration_minutes
- max_active_bookings_per_user
- requires_approval
- booking_window_days
- is_active
- created_at
- updated_at

---

### 6. resource_types
Типы ресурсов.

Основные поля:
- id
- category_id
- booking_rule_id
- name
- description
- is_active
- created_at
- updated_at

Связи:
- тип ресурса принадлежит категории;
- тип ресурса может быть связан с правилом бронирования.

---

### 7. resources
Экземпляры ресурсов.

Основные поля:
- id
- name
- description
- category_id
- type_id
- department_id
- status
- is_bookable
- is_active
- created_at
- updated_at

Связи:
- ресурс связан с категорией;
- ресурс связан с типом;
- ресурс может быть связан с подразделением.

---

### 8. resource_availability
Интервалы доступности ресурса.

Основные поля:
- id
- resource_id
- start_time
- end_time
- created_at
- updated_at

Связи:
- интервал доступности принадлежит ресурсу.

---

### 9. bookings
Бронирования ресурсов.

Основные поля:
- id
- resource_id
- user_id
- start_time
- end_time
- purpose
- status
- approved_by_user_id
- approved_at
- cancelled_at
- created_at
- updated_at

Связи:
- бронирование принадлежит ресурсу;
- бронирование принадлежит пользователю;
- подтверждение может быть связано с пользователем, принявшим решение.

---

### 10. candidates
Кандидаты.

Основные поля:
- id
- full_name
- email
- phone
- position
- status
- resume_link
- comment
- created_at
- updated_at

---

### 11. interview_stages
Этапы интервью.

Основные поля:
- id
- candidate_id
- interviewer_user_id
- stage_name
- scheduled_at
- status
- comment
- created_at
- updated_at

Связи:
- этап принадлежит кандидату;
- этап связан с интервьюером.

---

### 12. interview_feedback
Обратная связь по интервью.

Основные поля:
- id
- stage_id
- author_user_id
- decision
- comment
- created_at
- updated_at

Связи:
- обратная связь принадлежит этапу;
- обратная связь связана с автором.

---

### 13. audit_logs
Журнал действий пользователей.

Основные поля:
- id
- user_id
- entity_type
- entity_id
- action
- details
- created_at

Связи:
- запись журнала может быть связана с пользователем.

---

## Основные связи

Ключевые связи системы:
- `users` → `roles`
- `users` → `departments`
- `resource_types` → `resource_categories`
- `resource_types` → `booking_rules`
- `resources` → `resource_categories`
- `resources` → `resource_types`
- `resources` → `departments`
- `resource_availability` → `resources`
- `bookings` → `resources`
- `bookings` → `users`
- `candidates` → `interview_stages`
- `interview_stages` → `users`
- `interview_feedback` → `interview_stages`
- `interview_feedback` → `users`
- `audit_logs` → `users`

---

## Основные ограничения

Система должна учитывать следующие ограничения:
- email пользователя должен быть уникальным;
- пересекающиеся активные бронирования одного ресурса недопустимы;
- интервал доступности должен иметь корректный диапазон времени;
- интервалы бронирования должны иметь корректный диапазон времени;
- неактивные сущности не должны использоваться в рабочих операциях;
- статусы должны приниматься только из допустимого набора значений.

---

## Приоритет MVP

Для первой версии критически важны следующие таблицы:
- roles
- departments
- users
- resource_categories
- booking_rules
- resource_types
- resources
- resource_availability
- bookings

Таблицы, связанные с интервьюированием и журналированием, могут быть подключены позднее.