# API проекта ResourceFlow

## Назначение документа

Документ описывает общий подход к проектированию REST API и основные группы endpoint системы.

---

## Общие принципы API

API строится по REST-подходу и использует формат JSON для передачи данных.

Основные принципы:
- каждый endpoint относится к определенной сущности или процессу;
- методы HTTP используются в соответствии с назначением операции;
- ответы API должны иметь понятную и единообразную структуру;
- защищенные endpoint требуют аутентификации.

---

## Базовый путь API

```text
/api/v1
```

---

## Аутентификация

### Основные endpoint
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`

### Назначение
- вход пользователя;
- получение информации о текущем пользователе.

---

## Пользователи

### Основные endpoint
- `GET /api/v1/users`
- `GET /api/v1/users/{id}`
- `POST /api/v1/users`
- `PUT /api/v1/users/{id}`
- `DELETE /api/v1/users/{id}`

### Назначение
- просмотр списка пользователей;
- просмотр карточки пользователя;
- создание пользователя;
- редактирование пользователя;
- деактивация пользователя.

---

## Подразделения

### Основные endpoint
- `GET /api/v1/departments`
- `GET /api/v1/departments/{id}`
- `POST /api/v1/departments`
- `PUT /api/v1/departments/{id}`
- `DELETE /api/v1/departments/{id}`

---

## Категории ресурсов

### Основные endpoint
- `GET /api/v1/resource-categories`
- `POST /api/v1/resource-categories`
- `PUT /api/v1/resource-categories/{id}`
- `DELETE /api/v1/resource-categories/{id}`

---

## Правила бронирования

### Основные endpoint
- `GET /api/v1/booking-rules`
- `POST /api/v1/booking-rules`
- `PUT /api/v1/booking-rules/{id}`
- `DELETE /api/v1/booking-rules/{id}`

---

## Типы ресурсов

### Основные endpoint
- `GET /api/v1/resource-types`
- `POST /api/v1/resource-types`
- `PUT /api/v1/resource-types/{id}`
- `DELETE /api/v1/resource-types/{id}`

---

## Ресурсы

### Основные endpoint
- `GET /api/v1/resources`
- `GET /api/v1/resources/{id}`
- `POST /api/v1/resources`
- `PUT /api/v1/resources/{id}`
- `DELETE /api/v1/resources/{id}`

### Дополнительно
Могут поддерживаться параметры фильтрации:
- категория;
- тип;
- подразделение;
- статус;
- признак доступности для бронирования.

---

## Доступность ресурсов

### Основные endpoint
- `GET /api/v1/resources/{id}/availability`
- `POST /api/v1/resources/{id}/availability`
- `PUT /api/v1/resources/{id}/availability/{availabilityId}`
- `DELETE /api/v1/resources/{id}/availability/{availabilityId}`

---

## Бронирования

### Основные endpoint
- `GET /api/v1/bookings`
- `GET /api/v1/bookings/{id}`
- `POST /api/v1/bookings`
- `PUT /api/v1/bookings/{id}`
- `POST /api/v1/bookings/{id}/cancel`
- `POST /api/v1/bookings/{id}/approve`
- `POST /api/v1/bookings/{id}/reject`

### Назначение
- просмотр бронирований;
- создание бронирования;
- изменение бронирования;
- отмена бронирования;
- подтверждение или отклонение заявки.

---

## Кандидаты

### Основные endpoint
- `GET /api/v1/candidates`
- `GET /api/v1/candidates/{id}`
- `POST /api/v1/candidates`
- `PUT /api/v1/candidates/{id}`

---

## Этапы интервью

### Основные endpoint
- `GET /api/v1/interview-stages`
- `GET /api/v1/interview-stages/{id}`
- `POST /api/v1/interview-stages`
- `PUT /api/v1/interview-stages/{id}`

---

## Обратная связь по интервью

### Основные endpoint
- `POST /api/v1/interview-stages/{id}/feedback`
- `GET /api/v1/interview-stages/{id}/feedback`

---

## Журнал действий

### Основные endpoint
- `GET /api/v1/audit-logs`

---

## Отчетность

### Основные endpoint
- `GET /api/v1/reports/bookings`
- `GET /api/v1/reports/resources`
- `GET /api/v1/reports/interviews`

---

## Пример структуры успешного ответа

```json
{
  "success": true,
  "data": {}
}
```

---

## Пример структуры ответа с ошибкой

```json
{
  "success": false,
  "error": {
    "code": "validation_error",
    "message": "Некорректные входные данные"
  }
}
```

---

## Основные требования к API

API должно:
- корректно валидировать входные данные;
- возвращать понятные коды ошибок;
- учитывать права доступа пользователя;
- поддерживать единый формат ответов;
- быть пригодным для последовательной реализации по модулям.