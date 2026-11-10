---
trigger: always_on
description: โครงสร้างโปรเจกต์ สถาปัตยกรรม Clean Architecture มาตรฐานการเขียนโค้ด และรายละเอียด API ทั้งหมดของ horaveda-app
---

# Horaveda App — Project Context & Architecture Guide

เอกสารนี้สรุปโครงสร้างโปรเจกต์ `horaveda-app` (Go Backend API สำหรับระบบโหราศาสตร์ภารตะ/นิรายนะ Lahiri, การพยากรณ์พื้นดวงชะตาด้วย AnythingLLM และการยืนยันตัวตนด้วย Serial Key บน Supabase) เพื่อให้ AI Agent และนักพัฒนาเข้าใจบริบททั้งหมดของระบบทันทีโดยไม่ต้องสแกนไฟล์ทั้งหมดใหม่

---

## 1. Tech Stack & Environment
- **Language / Runtime**: Go (`go 1.24+`, module `horaveda-app`, ใช้ `net/http.ServeMux` มาตรฐาน)
- **Database & Auth**: Supabase (`github.com/supabase-community/supabase-go`, `gotrue-go`)
  - ตารางใน `public`: `users` (`id`, `username`), `serial_keys` (`id`, `key_code`, `is_used`, `user_id`)
  - ระบบ Auth: Supabase GoTrue (`auth.users`) โดยใช้อีเมลรูปแบบ `<username>@garbagecollection.info`
- **Authentication**: JWT (`github.com/golang-jwt/jwt/v5`, HS256) เก็บใน HttpOnly Cookie ชื่อ `auth_token` (และรองรับ Header `Authorization: Bearer <token>`)
- **Logging**: Uber Zap (`go.uber.org/zap`) + Daily Log Rotation (`pkg/logger/logger.go`)
  - หมุนเวียนไฟล์ Log รายวันอัตโนมัติ: `logs/app-YYYY-MM-DD.log`
  - บันทึกควบคู่ลง `logs/app.log` (รีเซ็ตทุกเที่ยงคืน) สำหรับดู Log ประจำวันล่าสุดได้อย่างสะดวก
  - หมุนเวียนไฟล์อัตโนมัติทุกเที่ยงคืน (Midnight Watcher 00:00:00) และตรวจสอบทุกครั้งที่มีการบันทึก
  - รองรับการคุมขนาด 100 MB ภายในวันเดียวกันด้วย `lumberjack.Logger` พร้อมบีบอัด `.gz`
  - กวาดล้างไฟล์ Log เก่าอายุเกิน 28 วันโดยอัตโนมัติ
  - แก้ไข Path ให้ชี้ไปยัง Project Root (`go.mod`) เสมอ ไม่กระจัดกระจายเมื่อรันเทสต์
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
  - `HORA_TRANSIT_API_URL`: URL ของบริการคำนวณดาวจร (Default: `http://localhost:8083/api/v1/hora/transitdasha_prompt`)
  - `ANYTHINGLLM_API_URL`: URL ของ AnythingLLM Developer API (Default: `http://localhost:3001/api/v1`)
  - `ANYTHINGLLM_API_KEY`: API Key ของ AnythingLLM
  - `ANYTHINGLLM_TIMEOUT_SECONDS`: เวลารอผลลัพธ์คำทำนายจาก AnythingLLM สูงสุด (Default: `120`)

---

## 2. Directory & File Structure (Clean Architecture)

```text
horaveda-app/
├── cmd/server/
│   └── main.go                                   # Entry point, Dependency Injection, HTTP Server & Graceful Shutdown
├── internal/
│   ├── config/
│   │   ├── config.go                             # โหลด .env, ตั้งค่า Config struct, สร้าง Supabase Client และตรวจเช็ค AnythingLLM Health
│   │   └── config_test.go
│   ├── domain/
│   │   └── model.go                              # Data Models: Register, Revoke, SerialKey, User, HoraPredict (Request & Response)
│   ├── repository/
│   │   ├── auth_repository.go                    # AuthRepository interface & Supabase Database / GoTrue Auth operations
│   │   └── auth_repository_test.go
│   ├── service/
│   │   ├── auth_service.go                       # AuthService: RegisterWithSerial (พร้อม Saga Rollback), RevokeUserAndSerial, VerifyToken, GetAllKeys
│   │   ├── auth_service_test.go
│   │   ├── hora_service.go                       # HoraService: GetHoraPrompt (ยิง POST ไปยัง Lahiri Prompt API :8083)
│   │   ├── hora_service_test.go
│   │   ├── anythingllm_service.go                # AnythingLLMService: จัดการ Workspace & Thread (1 User : 1 Workspace : 1 Thread), ตรวจสอบการมีอยู่, Auto-Recovery และส่ง Chat
│   │   ├── anythingllm_service_test.go
│   │   └── vedic_prompt.go                       # BuildVedicPredictionPrompt: ประกอบ Prompt พยากรณ์ตามหลักโหราศาสตร์พระเวท (Vedic Jyotish)
│   └── delivery/http/
│       ├── router.go                             # NewRouter: ลงทะเบียน Routes ทั้งหมดพร้อม Request Logging
│       ├── router_test.go
│       ├── middleware/
│       │   ├── auth_middleware.go                # RequireAuth (Cookie auth_token / Bearer JWT), RequireAdminKey (Constant-Time Compare), GetUserClaims
│       │   ├── cors.go                           # CORSMiddleware
│       │   └── middleware_test.go
│       └── handler/
│           ├── auth_handler.go                   # AuthHandler struct (ถือ AuthService, HoraService, AnythingLLMService, Logger, Cfg)
│           ├── register.go                       # HandleSerialRegister (POST /api/v1/auth/register-serial)
│           ├── verify.go                         # HandleVerifyAuth (GET /api/v1/auth/verify)
│           ├── logout.go                         # HandleLogout (POST /api/v1/auth/logout)
│           ├── revoke.go                         # HandleRevokeUser (POST & DELETE /api/v1/auth/revoke-user)
│           ├── hora.go                           # HandleHoraPredict (POST /api/v1/hora/predict, POST /api/v1/hora/prompt)
│           ├── hora_transit.go                   # HandleHoraTransitPredict (POST /api/v1/hora/transitpredict)
│           ├── debug.go                          # HandleDebugGetAllKeys
│           └── handler_test.go
├── pkg/
│   ├── logger/
│   │   ├── logger.go                             # NewLogger() พร้อม DailyRotateWriter (หมุนเวียนไฟล์รายวัน logs/app-YYYY-MM-DD.log + logs/app.log)
│   │   └── logger_test.go
│   └── response/
│       ├── response.go                           # Standard JSON Response: response.Success(w, code, msg, data), response.Error(w, code, msg)
│       └── errors.go                             # AppError struct & constructors (NewBadRequestError, NewUnauthorizedError, NewNotFoundError, NewInternalServerError, NewBadGatewayError)
├── logs/                                         # โฟลเดอร์เก็บไฟล์ Log รายวัน (app-YYYY-MM-DD.log และ app.log)
├── req/
│   ├── req.txt                                   # ตัวอย่าง Request / Response สำหรับทดสอบระบบ Horaveda Auth & Predict API
│   └── anythingllm_req.txt                       # ตัวอย่าง cURL สำหรับทดสอบ AnythingLLM Developer API โดยตรง
├── horaveda.postman_collection.json              # Postman Collection ของระบบ Horaveda
└── anythingllm.postman_collection.json           # Postman Collection สำหรับทดสอบ AnythingLLM API
```

---

## 3. AnythingLLM Integration Architecture (1 User : 1 Workspace : 1 Thread)

### หลักการทำงาน
1. **ความสัมพันธ์ของข้อมูล (1 User : 1 Workspace : 1 Shared Thread)**:
   - กำหนดให้ผู้ใช้ 1 บัญชี มี 1 Workspace บน AnythingLLM (`workspace_slug` คือ `strings.ToLower(username)`)
   - มี **1 Thread เริ่มต้นใช้งานร่วมกันทั้งพื้นดวงชะตากำเนิดและดาวจร** (`thread_slug` คือ `{workspace_slug}-hora`, Title: `พื้นดวงชะตา {username}`) เพื่อให้ AI ในห้องสนทนามีบริบทพื้นดวงกำเนิดอย่างต่อเนื่องในการทำนายดาวจร
2. **การตรวจสอบและใช้ซ้ำ (Workspace & Thread Deduplication)**:
   - ตรวจสอบใน AnythingLLM ก่อน (`GET /workspace/:slug` และ `GET /workspaces`)
   - **ถ้ามีอยู่แล้ว**: ใช้อันเดิมทันทีโดยเด็ดขาด **ไม่เรียก `POST /workspace/new` ซ้ำ** เพื่อป้องกัน AnythingLLM สุ่ม hash ต่อท้ายชื่อ (เช่น `testuser01-63141714`)
   - **ถ้าไม่มีในระบบ**: สร้าง Workspace ใหม่ (`POST /workspace/new`) และสร้าง Thread ใหม่ (`POST /workspace/{slug}/thread/new`)
3. **ระบบ Auto-Recovery (รับมือ 404 Not Found)**:
   - หากส่งข้อความพยากรณ์ (`POST .../chat`) แล้ว AnythingLLM ตอบกลับ 404 (เช่น Workspace ถูกลบออกจาก AnythingLLM ไปแล้ว)
   - ระบบจะตรวจจับด้วย `isNotFoundError`, ล้าง Cache ในหน่วยความจำ, สั่งสร้าง Workspace + Thread ใหม่ทันที และส่งข้อความพยากรณ์ซ้ำ (Retry) ให้อัตโนมัติ
4. **Graceful Fallback & การป้องกัน Cloudflare 502**:
   - เมื่อบริการ AnythingLLM ขัดข้องหรือไม่พร้อมใช้งาน ระบบจะไม่ส่ง HTTP 502 กลับไปยัง Reverse Proxy/Cloudflare (ซึ่งจะทำให้ Cloudflare ตัดหน้าแสดงผลเป็นหน้า HTML Error)
   - แต่จะส่งกลับเป็น HTTP 200 OK ที่ประกอบด้วยพิกัดดวงดาวจาก Lahiri และมีฟิลด์ `warning` แจ้งเตือนสถานะ เพื่อให้ Frontend แสดงผลข้อมูลผูกดวงได้อย่างต่อเนื่อง
5. **การประกอบ Prompt พยากรณ์ (Vedic Jyotish)**:
   - **พื้นดวงกำเนิด (`BuildVedicPredictionPrompt`)**: นำข้อมูลพิกัดดวงดาวกำเนิดมาประกอบเป็น Prompt ครอบคลุม สภาพลัคนา (Lagna), ตนุลัคน์, นักษัตร, เกณฑ์สายตา (Drishti), ระยะเชิงมุมดาวกุม (Conjunctions / Combust), ความเข้มแข็งของดาวผ่านนวางศ์จักร (D9) และกาลโยควิมโชตตรีทศา
   - **ดาวจรประจำเดือน (`BuildVedicTransitPredictionPrompt`)**: ผสานข้อมูลดาวชะตากำเนิด (Natal Chart) ร่วมกับข้อมูลดาวจรและทศา (Transit Chart & Dasha) สรุปอิทธิพลประจำเดือนในรูปแบบภาษาไทยที่เคร่งครัด ได้แก่ ภาพรวม, การงานและการศึกษา, การเงินและโชคลาภ, ความรักและสุขภาพ, 🪐 ดาวจรสำคัญ และ ✨ คำแนะนำการกระทำ (กรรม) และการทำบุญในกรอบพระพุทธศาสนา

---

## 4. API Endpoints Summary

| Method | Path | Protection | Handler | Description |
| :--- | :--- | :--- | :--- | :--- |
| `GET`, `POST` | `/api/v1/auth/serial-keys` | Public | `HandleGetSerialKeysStatus` | ตรวจสอบสถานะ Serial Key ทั้งหมดว่าคีย์ใดกำลังใช้งานอยู่ (`used`) หรือว่าง (`available`) โดยไม่ต้องใช้ Token |
| `POST` | `/api/v1/auth/register-serial` | Public | `HandleSerialRegister` | ลงทะเบียนด้วย `username` (`[a-zA-Z0-9]+`) และ `serial_key` พร้อมทำ Saga Rollback หากล้มเหลว และเซ็ต Cookie `auth_token` |
| `POST` | `/api/v1/auth/logout` | Public / Token | `HandleLogout` | ล้าง Cookie `auth_token` (`MaxAge: -1`) พร้อมถอนสิทธิ์ `serial_keys` (`is_used=false, user_id=nil`), ลบ `public.users` และลบ `auth.users` บน Supabase (ดึง `username` จาก JWT Token หรือ JSON Body) |
| `GET` | `/api/v1/auth/verify` | `RequireAuth` | `HandleVerifyAuth` | ตรวจสอบ JWT Token และคืนค่า `data: {"username": "..."}` |
| `POST`, `DELETE` | `/api/v1/auth/revoke-user` | `RequireAdminKey` | `HandleRevokeUser` | ถอนสิทธิ์ `serial_keys` (`is_used=false, user_id=nil`), ลบ `public.users` และลบ `auth.users` บน Supabase |
| `POST` | `/api/v1/hora/predict` (และ `/api/v1/hora/prompt`) | `RequireAuth` | `HandleHoraPredict` | รับ `{username, year, month, day, hour_thai, latitude, longitude}` คำนวณพิกัดดวงดาวผ่าน Lahiri แล้วส่งเข้า AnythingLLM (1 User : 1 Workspace : 1 Thread) คืนค่า `{status, prompt, prediction, workspace_slug, thread_slug}` |
| `POST` | `/api/v1/hora/transitpredict` | `RequireAuth` | `HandleHoraTransitPredict` | รับ `{year, month, day, hour_thai, time_zone, latitude, longitude, house_system, transit_year, transit_month}` คำนวณดาวชะตากำเนิดและดาวจรพร้อมทศาผ่าน Lahiri Concurrent calls แล้วส่งเข้า AnythingLLM ใน Thread ร่วมกัน `{workspace}-hora` คืนค่าคำทำนายประจำเดือนตามหลักพุทธศาสนา |
| `POST` | `/api/v1/hora/transitpredict_1` ถึง `/api/v1/hora/transitpredict_12` | `RequireAuth` | `HandleHoraTransitPredictMonth(1..12)` | **12 เส้นครอบรายเดือน ปี พ.ศ. 2570 (ค.ศ. 2027)** สำหรับหน้าเว็บเรียกทีละเดือน โดยระบบล็อกค่า `transit_year: 2027`, `transit_month: 1..12`, และ `house_system: "W"` ให้อัตโนมัติ |

---

## 5. Coding & Testing Conventions
1. **Standard Response Format (`pkg/response`)**:
   - สำเร็จ: `response.Success(w, http.StatusOK, "ข้อความภาษาไทย", data)` -> `{"success": true, "message": "...", "data": ...}`
   - ผิดพลาด: ใน Service ให้คืน `*response.AppError` และใน Handler ให้ใช้ `errors.As(err, &appErr)` แล้วเรียก `response.Error(w, appErr.Code, appErr.Message)`
2. **Comments & Language**:
   - คอมเมนต์อธิบายโค้ดและข้อความตอบกลับผู้ใช้ใช้ **ภาษาไทย** ที่ชัดเจน และรักษาคอมเมนต์เดิมที่มีอยู่เสมอ
3. **Unit Testing Rules (สำคัญมากเมื่อรันใน Sandbox บน Windows)**:
   - ห้ามใช้ `httptest.NewServer` ที่เปิดพอร์ต TCP จริงในการจำลอง External HTTP API เพราะ Sandbox บล็อกการ bind/connect socket
   - ให้ใช้ Custom `http.RoundTripper` (`roundTripFunc` ร่วมกับ `httptest.NewRecorder()`) แทน
4. **Documentation Sync**:
   - เมื่อเพิ่มหรือแก้ไข API Endpoint ให้อัปเดตตัวอย่างใน `req/req.txt`, `horaveda.postman_collection.json`, `req/anythingllm_req.txt` และ `anythingllm.postman_collection.json` ให้สอดคล้องกันเสมอ
