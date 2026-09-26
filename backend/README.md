# Backend - Personal Finance API

## Chạy local

```bash
cp .env.example .env
go mod tidy
go run .
```

API chạy tại `http://localhost:8080`. SQLite được tạo tự động theo `DATABASE_URL`.

Các endpoint chính: `POST /register`, `POST /login`, `GET /auth/google`, `POST/GET /transactions`, `DELETE /transactions/:id`, `GET /reports`.

Google OAuth cần điền `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` và đăng ký callback đúng với `GOOGLE_CALLBACK_URL`.
