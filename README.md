# 🪐 Horaveda App — Vedic Astrology & AI Prediction Backend API

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Architecture](https://img.shields.io/badge/Architecture-Clean%20Architecture-2ea44f?style=flat)](#architecture)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

**Horaveda App** คือระบบ Backend API พัฒนาด้วยภาษา Go ตามหลัก **Clean Architecture** สำหรับบริการคำนวณและพยากรณ์ดวงชะตาโหราศาสตร์ภารตะ/นิรายนะ (Vedic Sidereal / Lahiri Ayanamsa) ทั้งพื้นดวงชะตากำเนิด (Natal Chart) และดวงดาวจรรายเดือน (Monthly Transit & Vimshottari Dasha) ผสานการวิเคราะห์คำทำนายด้วย **AnythingLLM** และระบบยืนยันตัวตนด้วย **Serial Key** บน **Supabase**

---

## ✨ คุณสมบัติหลัก (Key Features)

- **🔐 ระบบยืนยันตัวตนด้วย Serial Key & Saga Rollback**:
  - ลงทะเบียนเข้าใช้งานด้วย `username` และ `serial_key` ผ่าน Supabase (`public.users`, `public.serial_keys` และ `auth.users`)
  - รองรับ **Saga Pattern Rollback** อัตโนมัติหากขั้นตอนใดล้มเหลวระหว่างการลงทะเบียนหรือถอนสิทธิ์
  - ออก JWT Token (HS256) รองรับทั้ง `HttpOnly Cookie` (`auth_token`) และ Header `Authorization: Bearer <token>`
  - เมื่อ `logout` ระบบจะล้าง Cookie พร้อมถอนสิทธิ์ `serial_key` คืนเป็นสถานะว่าง (`available`) และลบบัญชีผู้ใช้อัตโนมัติ
- **🪐 พยากรณ์พื้นดวงชะตากำเนิดและดาวจร (Vedic Jyotish & Lahiri)**:
  - คำนวณพิกัดดวงดาวกำเนิด (`POST /api/v1/hora/predict`) และดาวจรพร้อมกาลโยควิมโชตตรีทศาแบบ Concurrent (`POST /api/v1/hora/transitpredict`)
  - มี **12 เส้นครอบรายเดือนประจำปี พ.ศ. 2570 (ค.ศ. 2027)** (`POST /api/v1/hora/transitpredict_1` ถึง `transitpredict_12`) สำหรับหน้าเว็บเรียกแสดงผลทีละเดือน
- **🤖 AnythingLLM Integration (1 User : 1 Workspace : 1 Shared Thread)**:
  - จัดสรร 1 Workspace ต่อ 1 ผู้ใช้ และใช้ Thread ร่วมกัน (`{workspace}-hora`) เพื่อให้ AI มีบริบทพื้นดวงกำเนิดต่อเนื่องเมื่อทำนายดาวจรรายเดือน
  - ระบบ **Workspace Deduplication** (ถ้ามีอยู่แล้วใช้อันเดิม ถ้าไม่มีสร้างใหม่) และ **Auto-Recovery** เมื่อพบ `404 Not Found`
  - **Graceful Fallback (ป้องกัน Cloudflare 502)**: หากบริการ AI ขัดข้อง ระบบยังคงส่งคืน HTTP `200 OK` พร้อมข้อมูลพิกัดดวงดาวและข้อความ `warning`
- **📝 Daily Log Rotation**:
  - ใช้ Uber Zap ร่วมกับตัวหมุนเวียนไฟล์รายวัน (`logs/app-YYYY-MM-DD.log` และ `logs/app.log`) รีเซ็ตทุกเที่ยงคืน บีบอัดไฟล์ `.gz` และลบไฟล์เก่าเกิน 28 วันอัตโนมัติ

---

## 🏗️ โครงสร้างโปรเจกต์ (Clean Architecture) {#architecture}

```text
horaveda-app/
├── cmd/server/
│   └── main.go                                   # Entry point, Dependency Injection & Graceful Shutdown
├── internal/
│   ├── config/                                   # โหลดตัวแปรแวดล้อม (.env) และสร้าง Supabase Client
│   ├── domain/
│   │   └── model.go                              # Entities, DTOs และ Interfaces (AuthRepository, AuthUsecase, HoraUsecase, AnythingLLMUsecase)
│   ├── repository/
│   │   └── auth_repository.go                    # Supabase Database & GoTrue Auth Implementation
│   ├── service/
│   │   ├── auth_service.go                       # Business Logic ยืนยันตัวตน, Saga Rollback, ตรวจสอบสถานะ Serial Key
│   │   ├── hora_service.go                       # เชื่อมต่อ Lahiri Prompt API (:8083) แบบ Concurrent
│   │   ├── anythingllm_service.go                # จัดการ Workspace/Thread บน AnythingLLM (:3001)
│   │   └── vedic_prompt.go                       # ประกอบ Prompt โหราศาสตร์พระเวทและแนวทางพุทธศาสนา
│   └── delivery/http/
│       ├── router.go                             # HTTP ServeMux Router & Request Logging
│       ├── middleware/                           # RequireAuth (JWT), RequireAdminKey (Constant-Time), CORS
│       └── handler/                              # HTTP Handlers (Auth, Serial Keys, Natal & Transit Predict)
├── pkg/
│   ├── logger/                                   # Zap Logger + Daily Log Rotation (Midnight Watcher)
│   └── response/                                 # มาตรฐาน JSON Response & AppError
├── migrations/                                   # SQL Schema สำหรับตาราง users และ serial_keys
├── req/                                          # ตัวอย่าง HTTP Request / Response สำหรับทดสอบ
├── .env.example                                  # ตัวอย่างไฟล์ตั้งค่าตัวแปรแวดล้อม
└── Makefile                                      # คำสั่งลัดสำหรับ Build, Run, Test และ Coverage
```

---

## 🚀 การติดตั้งและเริ่มต้นใช้งาน (Getting Started)

### 1. ความต้องการของระบบ (Prerequisites)
- **Go** เวอร์ชัน `1.24` ขึ้นไป
- **Supabase Project** (สร้างตาราง `users` และ `serial_keys` ตามไฟล์ใน `migrations/`)
- **Lahiri Horoscope API** รันที่พอร์ต `8083` (ค่าเริ่มต้น)
- **AnythingLLM** เปิดโหมด Developer API ที่พอร์ต `3001` (ค่าเริ่มต้น)

### 2. ตั้งค่า Environment Variables
คัดลอกไฟล์ `.env.example` ไปเป็น `.env` และกรอกค่าคีย์ให้ครบถ้วน:

```bash
cp .env.example .env
```

### 3. ติดตั้ง Dependencies และรันเซิร์ฟเวอร์

```bash
# ดาวน์โหลดแพ็กเกจทั้งหมด
go mod download

# รันเซิร์ฟเวอร์ (เริ่มต้นที่พอร์ตตาม .env เช่น 8082)
go run cmd/server/main.go
```

หรือใช้สคริปต์ที่มีให้ในโปรเจกต์:
```bash
./run.sh
```

---

## 📡 สรุป API Endpoints

| Method | Endpoint | Auth | คำอธิบาย |
| :--- | :--- | :--- | :--- |
| `GET`, `POST` | `/api/v1/auth/serial-keys` | **Public** | ตรวจสอบสถานะ Serial Key ทั้งหมดว่าคีย์ใดใช้อยู่ (`used`) หรือว่าง (`available`) |
| `POST` | `/api/v1/auth/register-serial` | **Public** | ลงทะเบียนด้วย `username` และ `serial_key` พร้อมเซ็ต Cookie `auth_token` |
| `POST` | `/api/v1/auth/logout` | **Public / Token** | ล้าง Cookie `auth_token`, ถอนสิทธิ์ `serial_key` คืนสถานะว่าง และลบผู้ใช้ |
| `GET` | `/api/v1/auth/verify` | **RequireAuth** | ตรวจสอบความถูกต้องของ JWT Token และคืนค่า `username` |
| `POST`, `DELETE` | `/api/v1/auth/revoke-user` | **RequireAdminKey** | สำหรับผู้ดูแลระบบ (`X-Admin-Key`) ถอนสิทธิ์ Serial Key และลบผู้ใช้ |
| `POST` | `/api/v1/hora/predict` | **RequireAuth** | คำนวณและพยากรณ์พื้นดวงชะตากำเนิด (Natal Chart) |
| `POST` | `/api/v1/hora/transitpredict` | **RequireAuth** | คำนวณและพยากรณ์ดวงชะตาดาวจรประจำเดือน (กำหนดเดือนและปีจรเองได้) |
| `POST` | `/api/v1/hora/transitpredict_1` .. `_12` | **RequireAuth** | พยากรณ์ดวงชะตาดาวจรรายเดือน **ปี พ.ศ. 2570 (ค.ศ. 2027)** เดือนที่ 1 ถึง 12 |

> ดูตัวอย่าง Request / Response ฉบับเต็มได้ที่ [`req/req.txt`](./req/req.txt)

---

## 🧪 การรันชุดทดสอบ (Running Tests)

โปรเจกต์มี Unit Tests ครอบคลุมทุกแพ็กเกจ โดยจำลอง External HTTP API ผ่าน Custom `http.RoundTripper` (ไม่ต้องต่ออินเทอร์เน็ตหรือเปิดพอร์ตจริง):

```bash
# รัน Unit Tests ทั้งหมด
go test -v -count=1 ./...

# รันพร้อมตรวจสอบ Test Coverage
./test.sh
```

---

## 📄 สัญญาอนุญาต (License)

โปรเจกต์นี้เผยแพร่ภายใต้สัญญาอนุญาต [MIT License](./LICENSE) Copyright (c) 2026 Thanakrit Promsiri
