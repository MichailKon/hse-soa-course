# API Gateway Service

## Отвечает за
- Перенаправление запросов в другие микросервисы
- Ограничение запросов (ratelimit)
- Логирование всех запросов
- Преобразование запроса с фронта в формат, нужный для бека
- Проверка авторизации

## Границы:
- Ничего не делает сам -- перенаправляет на других

## API Endpoints
- POST /auth/login
- POST /auth/register
- POST /users/{id}
- GET /posts
- POST /posts
- PUT /posts/{id}
- DELETE /posts/{id}
- POST /posts/{id}/like
- POST /posts/{id}/comment
- POST /posts/{id}/comment/{comment_id}
