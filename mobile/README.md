# Flutter app

```bash
flutter pub get
flutter run --dart-define=API_URL=http://10.0.2.2:8080
flutter build apk --release --dart-define=API_URL=https://api.example.com
```

Trên iOS thay `10.0.2.2` bằng địa chỉ IP máy chạy backend trong mạng nội bộ. Cấu hình OAuth Google native cần thêm client ID vào Android/iOS theo tài liệu `google_sign_in`.
