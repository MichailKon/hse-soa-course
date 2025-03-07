ФИО: Кондрашин Михаил Юрьевич

Группа: 226

Вариант: Социальная сеть

HW2 examples:

Регистрация
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "misha567889",
    "email": "myukondrashin_1@edu.hse.ru",
    "password": "soa-course-bruh",
    "first_name": "Mikhail",
    "last_name": "Kondrashin",
    "birth_date": "2004-12-11T15:04:05Z",
    "phone_number": "+01234567890"
  }'
```

Логин
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "misha567889",
    "password": "soa-course-bruh"
  }'
```

Профиль
```bash
curl -X GET http://localhost:8080/api/users/profile \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>"
```

```bash
curl -X PUT http://localhost:8080/api/users/profile \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "first_name": "Bruh"    
  }'
```