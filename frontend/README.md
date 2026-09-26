# Frontend - Finora

Ứng dụng React chạy trực tiếp bằng CDN, không cần Node.js. Khởi động backend trước rồi mở `index.html` bằng trình duyệt. Khi trình duyệt chặn file local, chạy `python -m http.server 5500` trong thư mục `frontend` và truy cập `http://localhost:5500`.

API mặc định là `http://localhost:8080`. Có thể đổi bằng `localStorage.setItem('finance_api', 'https://api-cua-ban.example')` trong console trình duyệt.
