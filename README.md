# Bank API

REST API банковского сервиса на Go.

Проект реализован по техническому заданию: регистрация и аутентификация пользователей, JWT, банковские счета, карты, переводы, кредиты, аналитика, прогноз баланса, интеграция с ЦБ РФ, SMTP-уведомления, шифрование и хеширование чувствительных данных. 

---

## Стек

- Go 1.23+
- PostgreSQL 17
- Docker Compose
- gorilla/mux
- lib/pq
- golang-jwt/jwt/v5
- bcrypt
- HMAC-SHA256
- PostgreSQL pgcrypto
- logrus
- gomail.v2
- beevik/etree

---

## Возможности API

Реализовано:

- регистрация пользователей;
- проверка уникальности `email` и `username`;
- аутентификация через JWT;
- срок жизни JWT — 24 часа;
- создание банковских счетов;
- просмотр счетов пользователя;
- пополнение счёта;
- списание средств;
- переводы между счетами;
- выпуск виртуальных карт;
- оплата картой;
- генерация номера карты по алгоритму Луна;
- шифрование номера и срока карты через PostgreSQL `pgcrypto`;
- bcrypt-хеширование CVV;
- HMAC-SHA256 для контроля целостности данных карты;
- проверка HMAC при чтении карты и при оплате картой;
- оформление кредита;
- получение ключевой ставки ЦБ РФ через SOAP;
- расчёт аннуитетного платежа;
- создание графика платежей;
- прогноз баланса с учётом будущих платежей;
- аналитика доходов, расходов и кредитной нагрузки;
- шедулер обработки платежей;
- SMTP-интеграция для уведомлений;
- отправка email-уведомлений после платёжных операций.

---

## Структура проекта

bank-api/
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   └── postgres.go
│   ├── httpapi/
│   │   ├── handler.go
│   │   ├── middleware.go
│   │   └── router.go
│   ├── model/
│   │   └── model.go
│   ├── repository/
│   │   ├── errors.go
│   │   └── store.go
│   ├── scheduler/
│   │   └── scheduler.go
│   ├── security/
│   │   ├── card.go
│   │   ├── card_test.go
│   │   ├── hmac.go
│   │   ├── jwt.go
│   │   └── password.go
│   └── service/
│       ├── annuity_test.go
│       ├── auth.go
│       ├── bank.go
│       ├── cbr.go
│       ├── email.go
│       └── errors.go
│
├── migrations/
│   └── 001_init.sql
│
├── .env.example
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md

---

## Переменные окружения

Пример файла .env.example:

APP_PORT=8080
DATABASE_URL=postgres://bank:bank@127.0.0.1:5433/bank?sslmode=disable

JWT_SECRET=my-super-secret-jwt-key-1234567890
CARD_HMAC_SECRET=my-super-secret-hmac-key-123456789
CARD_PGP_KEY=my-super-secret-pgp-key-1234567890

LOG_LEVEL=debug
SCHEDULER_INTERVAL=12h

SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@bank.local

CBR_MARGIN_PERCENT=5

---

## PostgreSQL

База данных запускается через Docker Compose.

Файл docker-compose.yml использует порт `5433` на хосте:

```yaml
services:
  postgres:
    image: postgres:17
    container_name: bank-postgres
    environment:
      POSTGRES_USER: bank
      POSTGRES_PASSWORD: bank
      POSTGRES_DB: bank
    ports:
      - "5433:5432"
    volumes:
      - bank_pg_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d:ro

volumes:
  bank_pg_data:


Причина использования порта `5433`: на локальной машине порт `5432` может быть занят другим PostgreSQL.

---

## Запуск проекта

### 1. Запустить PostgreSQL

  powershell
docker compose up -d


Проверка контейнера:

   powershell
docker ps


Ожидаемый порт:

   text
0.0.0.0:5433->5432/tcp


Проверка подключения к PostgreSQL:

   powershell
docker exec -e PGPASSWORD=bank -it bank-postgres psql -h 127.0.0.1 -U bank -d bank -c "select current_user;"


Ожидаемый результат:

   text
 current_user
--------------
 bank


---

### 2. Установить зависимости

   powershell
go mod tidy


---

### 3. Запустить API

   powershell
go run ./cmd/api


Успешный запуск:

   json
{"addr":":8080","level":"info","msg":"http server started"}


Также запускается шедулер платежей:

   json
{"level":"info","msg":"scheduled payments processed","processed":0}


---

## Эндпоинты

### Публичные
```
| Метод | URL | Назначение |
|      |      |            |
| POST | `/register` | регистрация пользователя |
| POST | `/login` | аутентификация пользователя |
```
### Защищённые

Для защищённых эндпоинтов нужен заголовок:

   http
Authorization: Bearer <JWT_TOKEN>
```

| Метод | URL | Назначение |
|       |     |            |
| POST | `/accounts` | создать счёт |
| GET | `/accounts` | получить счета пользователя |
| POST | `/accounts/{accountId}/deposit` | пополнить счёт |
| POST | `/accounts/{accountId}/withdraw` | списать со счёта |
| POST | `/transfer` | перевести деньги между счетами |
| POST | `/cards` | выпустить карту |
| GET | `/cards` | получить карты пользователя |
| POST | `/cards/{cardId}/pay` | оплатить картой |
| POST | `/credits` | оформить кредит |
| GET | `/credits/{creditId}/schedule` | получить график платежей |
| GET | `/analytics` | получить аналитику |
| GET | `/accounts/{accountId}/predict?days=N` | прогноз баланса |

---

## Проверка API

Ниже приведены реальные сценарии, которые были проверены вручную.

---

### 1. Проверка, что сервер слушает порт 8080

   powershell
netstat -ano | findstr :8080


Результат:

   text
TCP    0.0.0.0:8080    0.0.0.0:0    LISTENING
TCP    [::]:8080       [::]:0       LISTENING


---

### 2. Регистрация пользователя

PowerShell иногда искажает JSON при использовании `curl`, поэтому использовался вариант с `--%`.

```powershell
curl.exe --% -X POST http://localhost:8080/register -H "Content-Type: application/json" -d "{\"email\":\"user@example.com\",\"username\":\"user1\",\"password\":\"StrongPass123\"}"


Результат:

  json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "user1",
    "created_at": "2026-05-06T11:16:48.614263Z"
  },
  "token": "JWT_TOKEN"
}


После регистрации был получен JWT-токен.

---

### 3. Сохранение JWT в переменную PowerShell

   powershell
$token = "JWT_TOKEN"
```

---

### 4. Создание счёта

   powershell
Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8080/accounts" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
id            : 1
user_id       : 1
currency      : RUB
balance_cents : 0
balance       : 0


---

### 5. Пополнение счёта

  powershell
curl.exe --% -X POST http://localhost:8080/accounts/1/deposit -H "Authorization: Bearer JWT_TOKEN" -H "Content-Type: application/json" -d "{\"amount\":10000}"


Результат:

   text
id            : 1
user_id       : 1
currency      : RUB
balance_cents : 1000000
balance       : 10000


---

### 6. Просмотр счетов

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/accounts" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
id            : 1
user_id       : 1
currency      : RUB
balance_cents : 1000000
balance       : 10000


---

### 7. Выпуск карты

   powershell
$body = @{
  account_id = 1
} | ConvertTo-Json

Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8080/cards" `
  -Headers @ { Authorization = "Bearer $token"} `
  -ContentType "application/json" `
  -Body $body


Результат:

text
id         : 1
user_id    : 1
account_id : 1
last4      : 4075
created_at : 2026-05-06T11:25:14.051218Z


---

### 8. Просмотр карт

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/cards" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
id         : 1
user_id    : 1
account_id : 1
number     : 4000002552694075
expiry     : 05/29
last4      : 4075
created_at : 2026-05-06T11:25:14.051218Z


Карта была успешно сохранена и прочитана из БД.  
Номер и срок карты хранятся в зашифрованном виде через `pgcrypto`.  
CVV хранится как bcrypt-хеш.
При чтении карты выполняется HMAC-проверка целостности расшифрованных карточных данных.

---

### 8.1. Оплата картой

Оплата картой на `700` рублей:

   powershell
$body = @{
  amount = 700
} | ConvertTo-Json

Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8080/cards/1/pay" `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body $body


Результат:

   text
id            : 1
user_id       : 1
currency      : RUB
balance_cents : 930000
balance       : 9300


Проверка расчёта:

   text
баланс до оплаты:    10000
сумма оплаты:          700
баланс после оплаты:  9300


При оплате картой выполняются:

- проверка JWT;
- проверка принадлежности карты пользователю;
- проверка HMAC целостности карточных данных;
- проверка достаточности средств;
- списание средств со связанного счёта;
- создание транзакции типа `card_payment`;
- вызов SMTP-компонента для отправки email-уведомления.

---

### 9. Аналитика после пополнения

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/analytics" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
month_income_cents  : 1000000
month_income        : 10000
month_expense_cents : 0
month_expense       : 0
credit_load_cents   : 0
credit_load         : 0
---

### 9.1. Аналитика после оплаты картой

После оплаты картой на `700` рублей аналитика показывает расход:

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/analytics" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
month_income_cents  : 1000000
month_income        : 10000
month_expense_cents : 70000
month_expense       : 700
credit_load_cents   : 0
credit_load         : 0


Оплата картой корректно учитывается как расход.


---

### 10. Прогноз баланса до кредита

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/accounts/1/predict?days=30" `
  -Headers @{ Authorization = "Bearer $token" }
 

Результат:

   text
account_id            : 1
days                  : 30
current_balance_cents : 1000000
current_balance       : 10000
planned_debits_cents  : 0
planned_debits        : 0
predicted_cents       : 1000000
predicted             : 10000


---

### 11. Создание второго счёта

   powershell
Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8080/accounts" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
id            : 2
user_id       : 1
currency      : RUB
balance_cents : 0
balance       : 0


---

### 12. Перевод между счетами

Перевод `1500` рублей со счёта `1` на счёт `2`.

   powershell
$body = @{
  from_account_id = 1
  to_account_id = 2
  amount = 1500
} | ConvertTo-Json

Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8080/transfer" `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body $body


Результат:

   text
status : ok


---

### 13. Проверка балансов после перевода

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/accounts" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
account 1 balance: 8500
account 2 balance: 1500


---

### 14. Аналитика после перевода

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/analytics" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

  text
month_income_cents  : 1150000
month_income        : 11500
month_expense_cents : 150000
month_expense       : 1500
credit_load_cents   : 0
credit_load         : 0


---

### 15. Оформление кредита

Оформление кредита на `5000` рублей на срок `6` месяцев.

powershell
$body = @{
  account_id = 1
  amount = 5000
  term_months = 6
} | ConvertTo-Json

Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8080/credits" `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body $body


Результат:

   text
id                    : 1
user_id               : 1
account_id            : 1
principal_cents       : 500000
principal             : 5000
annual_rate           : 19,5
term_months           : 6
monthly_payment_cents : 88137
monthly_payment       : 881,37
status                : active
```

Кредит успешно оформлен.  
Ставка получена через интеграцию с ЦБ РФ и увеличена на маржу из переменной `CBR_MARGIN_PERCENT`.

---

### 16. График платежей по кредиту

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/credits/1/schedule" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
id            : 1
credit_id     : 1
account_id    : 1
due_date      : 2026-06-06T00:00:00Z
amount_cents  : 88137
amount        : 881,37
penalty_cents : 0
penalty       : 0
status        : pending

id            : 2
credit_id     : 1
account_id    : 1
due_date      : 2026-07-06T00:00:00Z
amount_cents  : 88137
amount        : 881,37
penalty_cents : 0
penalty       : 0
status        : pending

id            : 3
credit_id     : 1
account_id    : 1
due_date      : 2026-08-06T00:00:00Z
amount_cents  : 88137
amount        : 881,37
penalty_cents : 0
penalty       : 0
status        : pending

id            : 4
credit_id     : 1
account_id    : 1
due_date      : 2026-09-06T00:00:00Z
amount_cents  : 88137
amount        : 881,37
penalty_cents : 0
penalty       : 0
status        : pending

id            : 5
credit_id     : 1
account_id    : 1
due_date      : 2026-10-06T00:00:00Z
amount_cents  : 88137
amount        : 881,37
penalty_cents : 0
penalty       : 0
status        : pending

id            : 6
credit_id     : 1
account_id    : 1
due_date      : 2026-11-06T00:00:00Z
amount_cents  : 88137
amount        : 881,37
penalty_cents : 0
penalty       : 0
status        : pending
```

---

### 17. Проверка баланса после оформления кредита

После оформления кредита баланс счёта `1` увеличился:

   text
до кредита:     8500
сумма кредита:  5000
после кредита: 13500


Проверка:

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/accounts" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
account 1 balance: 13500
account 2 balance: 1500


---

### 18. Прогноз баланса после кредита

   powershell
Invoke-RestMethod `
  -Method GET `
  -Uri "http://localhost:8080/accounts/1/predict?days=60" `
  -Headers @{ Authorization = "Bearer $token" }


Результат:

   text
account_id            : 1
days                  : 60
current_balance_cents : 1350000
current_balance       : 13500
planned_debits_cents  : 88137
planned_debits        : 881,37
predicted_cents       : 1261863
predicted             : 12618,63


Прогноз учитывает ближайший платёж по кредиту.

---

## Финальная проверка кода

Перед сдачей были выполнены команды:

   powershell
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/api


Результат тестов:

   text
?       bank-api/cmd/api              [no test files]
?       bank-api/internal/config      [no test files]
?       bank-api/internal/db          [no test files]
?       bank-api/internal/httpapi     [no test files]
?       bank-api/internal/model       [no test files]
?       bank-api/internal/repository  [no test files]
?       bank-api/internal/scheduler   [no test files]
ok      bank-api/internal/security
ok      bank-api/internal/service


Команда:

   powershell
go vet ./...


прошла без ошибок.

Команда:

   powershell
go build ./cmd/api


прошла успешно.  
После сборки появился файл `api.exe`.

---

## Сброс базы данных

Для полного пересоздания базы:

   powershell
docker compose down -v
docker compose up -d


Эта команда удаляет volume PostgreSQL и заново применяет миграции из папки `migrations`.

---

## Безопасность

В проекте реализованы следующие меры безопасности:

- пароли пользователей хешируются через bcrypt;
- CVV карты хешируется через bcrypt;
- номер карты и срок действия шифруются через PostgreSQL `pgcrypto`;
- HMAC-SHA256 используется для контроля целостности карточных данных при чтении карты и при  оплате картой;
- авторизация защищённых эндпоинтов выполняется через JWT;
- JWT-секрет хранится в переменной окружения;
- SQL-запросы параметризованы;
- доступ к счетам и картам проверяется по `user_id`;
- оплата картой доступна только владельцу карты;
- при оплате картой проверяется баланс связанного счёта;
- операции оплаты картой сохраняются в истории транзакций как `card_payment`.


## Команды для отправки на GitHub

