---
trigger: always_on
description: โครงสร้างโปรเจกต์ สถาปัตยกรรม Clean Architecture มาตรฐานการเขียนโค้ด และรายละเอียด API ทั้งหมดของ horaveda-app
---

# Horaveda App — Project Context & Architecture Guide

เอกสารนี้สรุปโครงสร้างโปรเจกต์ `horaveda-app` (Go Backend API สำหรับระบบโหราศาสตร์ภารตะ/นิรายนะ Lahiri และการยืนยันตัวตนด้วย Serial Key บน Supabase) เพื่อให้ AI Agent เข้าใจบริบททั้งหมดของระบบทันทีโดยไม่ต้องสแกนไฟล์ทั้งหมดใหม่เมื่อเริ่มแชทใหม่

---

## 1. Tech Stack & Environment
- **Language / Runtime**: Go (`go 1.24.4`, module `horaveda-app`, ใช้ `net/http.ServeMux` มาตรฐาน)
- **Database & Auth**: Supabase (`github.com/supabase-community/supabase-go`, `gotrue-go`)
  - ตารางใน `public`: `users` (`id`, `username`), `serial_keys` (`id`, `key_code`, `is_used`, `user_id`)
  - ระบบ Auth: Supabase GoTrue (`auth.users`) โดยใช้อีเมลรูปแบบ `<username>@garbagecollection.info`
- **Authentication**: JWT (`github.com/golang-jwt/jwt/v5`, HS256) เก็บใน HttpOnly Cookie ชื่อ `auth_token` (และรองรับ Header `Authorization: Bearer <token>`)
- **Logging**: Uber Zap (`go.uber.org/zap`) + Lumberjack log rotation (`pkg/logger/logger.go`)
- **Environment Variables (`.env` / `internal/config/config.go`)**:
  - `NEXT_PUBLIC_SUPABASE_URL`: URL ของ Supabase
  - `NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY`: Service Role Key (Bypass RLS)
  - `NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY`: Publishable/Anon Key (Fallback)
  - `PORT`: พอร์ตของเซิร์ฟเวอร์ (Default: `8080`, ใน `.env` ตั้งไว้ `8082`)
  - `CORS_ORIGIN`: โดเมนที่อนุญาต CORS (Default: `https://hora.garbagecollection.info`)
  - `JWT_SECRET`: คีย์สำหรับลงนาม JWT
  - `COOKIE_SECURE`: ตั้งค่า `Secure` flag ของ Cookie (`true` หรือเมื่อ `APP_ENV=production`)
  - `ADMIN_API_KEY`: คีย์สำหรับเรียกใช้ Admin Endpoints (ผ่าน Header `X-Admin-Key` หรือ `Authorization: Bearer`)
  - `HORA_API_URL`: URL ของบริการคำนวณพื้นดวงชะตา Lahiri (Default: `http://localhost:8083/api/v1/hora/prompt`)

---

## 2. Directory & File Structure (Clean Architecture)

```text
horaveda-app/
├── cmd/server/main.go                            # Entry point, Dependency Injection, HTTP Server & Graceful Shutdown
├── internal/
│   ├── config/
│   │   ├── config.go                             # โหลด .env, ตั้งค่า Config struct และสร้าง Supabase Client
│   │   └── config_test.go
│   ├── domain/
│   │   └── model.go                              # Structs: RegisterRequest, RevokeUserRequest, RevokeUserResponse, SerialKey, User, HoraPredictRequest, HoraCalculateReq, HoraPromptRequest, HoraPromptResponse
│   ├── repository/
│   │   ├── auth_repository.go                    # AuthRepository interface & Supabase DB/Auth operations
│   │   └── auth_repository_test.go
│   ├── service/
│   │   ├── auth_service.go                       # AuthService: RegisterWithSerial (พร้อม Saga Rollback), RevokeUserAndSerial, VerifyToken, GetAllKeys
│   │   ├── auth_service_test.go
│   │   ├── hora_service.go                       # HoraService: GetHoraPrompt (ยิง POST ไปยัง Lahiri Prompt API :8083)
│   │   └── hora_service_test.go
│   └── delivery/http/
│       ├── router.go                             # NewRouter: ลงทะเบียน Routes ทั้งหมดพร้อม Request Logging
│       ├── router_test.go
│       ├── middleware/
│       │   ├── auth_middleware.go                # RequireAuth (Cookie auth_token / Bearer JWT), RequireAdminKey (Constant-Time Compare), GetUserClaims
│       │   ├── cors.go                           # CORSMiddleware
│       │   └── middleware_test.go
│       └── handler/
│           ├── auth_handler.go                   # AuthHandler struct (ถือ AuthService, HoraService, Logger, Cfg)
│           ├── register.go                       # HandleSerialRegister (POST /api/v1/auth/register-serial)
│           ├── verify.go                         # HandleVerifyAuth (GET /api/v1/auth/verify)
│           ├── logout.go                         # HandleLogout (POST /api/v1/auth/logout)
│           ├── revoke.go                         # HandleRevokeUser (POST & DELETE /api/v1/auth/revoke-user)
│           ├── hora.go                           # HandleHoraPredict (POST /api/v1/hora/predict, POST /api/v1/hora/prompt)
│           ├── debug.go                          # HandleDebugGetAllKeys
│           └── handler_test.go
├── pkg/
│   ├── logger/logger.go                          # NewLogger() คืนค่า *zap.SugaredLogger
│   └── response/
│       ├── response.go                           # Standard JSON Response: response.Success(w, code, msg, data), response.Error(w, code, msg)
│       └── errors.go                             # AppError struct & constructors (NewBadRequestError, NewUnauthorizedError, NewNotFoundError, NewInternalServerError)
├── req/req.txt                                   # ตัวอย่าง Request / Response สำหรับทดสอบ API
└── horaveda.postman_collection.json              # Postman Collection ของโปรเจกต์
```

---

## 3. API Endpoints Summary

| Method | Path | Protection | Handler | Description |
| :--- | :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/auth/register-serial` | Public | `HandleSerialRegister` | ลงทะเบียนด้วย `username` (เฉพาะ `[a-zA-Z0-9]+`) และ `serial_key` พร้อมทำ Saga Rollback หากล้มเหลว และเซ็ต Cookie `auth_token` |
| `POST` | `/api/v1/auth/logout` | Public | `HandleLogout` | ล้าง Cookie `auth_token` (`MaxAge: -1`) |
| `GET` | `/api/v1/auth/verify` | `RequireAuth` | `HandleVerifyAuth` | ตรวจสอบ JWT Token และคืนค่า `data: {"username": "..."}` |
| `POST`, `DELETE` | `/api/v1/auth/revoke-user` | `RequireAdminKey` | `HandleRevokeUser` | ถอนสิทธิ์ `serial_keys` (`is_used=false, user_id=nil`), ลบ `public.users` และลบ `auth.users` บน Supabase |
| `POST` | `/api/v1/hora/predict` (และ `/api/v1/hora/prompt`) | `RequireAuth` | `HandleHoraPredict` | รับ `{year, month, day, hour_thai, latitude, longitude}` ยิงต่อไปยัง `HORA_API_URL` (`time_zone: 7.0`, `house_system: "W"`) และคืนค่า `prompt` |

---

## 4. Coding & Testing Conventions
1. **Standard Response Format (`pkg/response`)**:
   - สำเร็จ: เรียกใช้ `response.Success(w, http.StatusOK, "ข้อความภาษาไทย", data)` จะได้ JSON `{"success": true, "message": "...", "data": ...}`
   - ผิดพลาด: ใน Service ให้คืน `*response.AppError` และใน Handler ให้ใช้ `errors.As(err, &appErr)` แล้วเรียก `response.Error(w, appErr.Code, appErr.Message)`
2. **Comments & Language**:
   - คอมเมนต์อธิบายโค้ดและข้อความตอบกลับผู้ใช้ใช้ **ภาษาไทย** ที่ชัดเจน และรักษาคอมเมนต์เดิมที่มีอยู่เสมอ
3. **Unit Testing Rules (สำคัญมากเมื่อรันใน Sandbox บน Windows)**:
   - ห้ามใช้ `httptest.NewServer` ที่เปิดพอร์ต TCP จริงในการจำลอง External HTTP API เพราะ Sandbox บล็อกการ bind/connect socket
   - ให้ใช้ Custom `http.RoundTripper` (`roundTripFunc` ร่วมกับ `httptest.NewRecorder()`) แทน (ดูตัวอย่างใน `internal/service/hora_service_test.go` และ `internal/repository/auth_repository_test.go`)
4. **Documentation Sync**:
   - เมื่อเพิ่มหรือแก้ไข API Endpoint ให้อัปเดตตัวอย่างใน `req/req.txt` และ `horaveda.postman_collection.json` ให้สอดคล้องกันเสมอ
