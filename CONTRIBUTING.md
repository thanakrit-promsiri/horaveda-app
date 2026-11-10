# แนวทางการมีส่วนร่วมพัฒนา (Contributing Guide)

ขอบคุณที่สนใจร่วมพัฒนาโปรเจกต์ **Horaveda App**! เอกสารนี้ระบุมาตรฐานการเขียนโค้ด การทดสอบ และขั้นตอนการส่ง Pull Request เพื่อให้โค้ดเบสมีความสะอาดและดูแลรักษาง่ายตามหลัก **Clean Architecture**

---

## 1. มาตรฐานสถาปัตยกรรม (Clean Architecture Rules)

1. **Domain Layer (`internal/domain/`)**:
   - เก็บ Structs, Entities, DTOs และ Interfaces (`AuthRepository`, `AuthUsecase`, `HoraUsecase`, `AnythingLLMUsecase`)
   - **ห้าม** import แพ็กเกจภายนอกหรือแพ็กเกจชั้น `repository`, `service`, `delivery`
2. **Repository Layer (`internal/repository/`)**:
   - ทำหน้าที่เชื่อมต่อฐานข้อมูล Supabase และระบบภายนอกตามสัญญาใน `domain`
3. **Service / Usecase Layer (`internal/service/`)**:
   - เก็บ Business Logic การลงทะเบียน, Saga Rollback, การคำนวณดวงชะตา และการประกอบ Prompt
   - พึ่งพา `domain.AuthRepository` ผ่าน Interface เท่านั้น
4. **Delivery Layer (`internal/delivery/http/`)**:
   - จัดการ HTTP Routing, Middleware (`RequireAuth`, `RequireAdminKey`, `CORSMiddleware`) และแปลงข้อมูล JSON Request/Response
   - ใช้ `pkg/response` สำหรับส่ง JSON มาตรฐาน (`response.Success`, `response.Error`)

---

## 2. มาตรฐานการเขียนโค้ดและคอมเมนต์

- ใช้ **ภาษาไทย** สำหรับคอมเมนต์อธิบาย Business Logic และข้อความตอบกลับผู้ใช้ (`message` ใน JSON Response)
- รักษาคอมเมนต์เดิมที่มีอยู่เสมอหากไม่เกี่ยวข้องกับการแก้ไข
- จัดรูปแบบโค้ดด้วย `gofmt` และตรวจสอบด้วย `go vet ./...` ก่อน Commit เสมอ

---

## 3. มาตรฐานการเขียน Unit Tests

- ทุกฟีเจอร์หรือการแก้ไขบั๊กควรมี Unit Test กำกับในไฟล์ `*_test.go`
- **สำหรับการจำลอง External HTTP API**: ให้ใช้ Custom `http.RoundTripper` (`roundTripFunc` ร่วมกับ `httptest.NewRecorder()`) แทนการเปิดพอร์ต TCP จริงด้วย `httptest.NewServer` เพื่อให้รันในสภาพแวดล้อม Sandbox หรือ CI ได้อย่างเสถียร
- รันชุดทดสอบทั้งหมดด้วยคำสั่ง:
  ```bash
  go test -v -count=1 ./...
  ```

---

## 4. การอัปเดตเอกสารควบคู่กับ API

เมื่อมีการเพิ่มหรือแก้ไข API Endpoint กรุณาอัปเดตไฟล์ต่อไปนี้ให้ตรงกันเสมอ:
- `req/req.txt`
- `horaveda.postman_collection.json`
- `README.md` และ `GEMINI.md`
