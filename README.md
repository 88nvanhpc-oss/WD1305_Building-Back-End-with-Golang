# Finora - Ứng dụng quản lý tài chính cá nhân

Ứng dụng gồm backend Golang, giao diện ReactJS và ứng dụng Flutter. Người dùng có thể đăng ký/đăng nhập, ghi nhận thu nhập hoặc chi tiêu, xóa giao dịch và xem báo cáo tổng hợp.

## Cấu trúc

```text
backend/    API RESTful Gin, SQLite, JWT, bcrypt, Google OAuth
frontend/   ReactJS + Recharts chạy qua CDN
mobile/     Flutter + Provider + google_sign_in
```

## Chạy backend

Yêu cầu Go 1.22 trở lên.

```bash
cd backend
copy .env.example .env
go mod tidy
go run .
```

Backend mặc định chạy ở `http://localhost:8080`. SQLite được tạo tự động thành `finance.db`. Các biến môi trường nằm trong `backend/.env.example`.

Để bật đăng nhập Google, tạo OAuth Client trên Google Cloud, điền `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` và đặt callback URL là `http://localhost:8080/auth/google/callback`.

## Chạy web

Mở terminal thứ hai:

```bash
cd frontend
python -m http.server 5500
```

Truy cập `http://localhost:5500`. Frontend dùng React, ReactDOM, Babel và Recharts từ `cdn.jsdelivr.net`, không cần cài Node.js.

## Chạy Flutter

```bash
cd mobile
flutter pub get
flutter run --dart-define=API_URL=http://10.0.2.2:8080
```

Build APK:

```bash
flutter build apk --release --dart-define=API_URL=https://api.example.com
```

Trên thiết bị thật, thay địa chỉ API bằng IP LAN của máy chạy backend. Để dùng Google Sign-In native, cấu hình Android OAuth client và iOS URL scheme theo tài liệu gói `google_sign_in`.

## API

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| POST | `/register` | Tạo tài khoản email/mật khẩu |
| POST | `/login` | Đăng nhập và nhận JWT |
| GET | `/auth/google` | Bắt đầu Google OAuth |
| GET | `/auth/google/callback` | Callback Google |
| GET | `/me` | Lấy người dùng hiện tại |
| POST | `/transactions` | Thêm giao dịch |
| GET | `/transactions` | Danh sách giao dịch |
| DELETE | `/transactions/:id` | Xóa giao dịch |
| GET | `/reports` | Tổng thu nhập, chi tiêu, số dư |

Các endpoint giao dịch, báo cáo và `/me` yêu cầu header `Authorization: Bearer <token>`.
