## 📌 สรุปการเปลี่ยนแปลง (Description)
<!-- อธิบายสั้นๆ ว่า Pull Request นี้เพิ่มฟีเจอร์ แก้ไขบั๊ก หรือปรับปรุงส่วนใด -->

## 🛠️ ประเภทของการเปลี่ยนแปลง (Type of Change)
- [ ] ✨ ฟีเจอร์ใหม่ (New feature)
- [ ] 🐛 แก้ไขบั๊ก (Bug fix)
- [ ] ♻️ ปรับโครงสร้างโค้ด (Refactoring / Clean Architecture)
- [ ] 📝 อัปเดตเอกสาร (Documentation update)

## ✅ รายการตรวจสอบก่อนส่ง (Checklist)
- [ ] รัน `go test -count=1 ./...` ผ่านครบทุกแพ็กเกจ
- [ ] รัน `go vet ./...` ไม่พบข้อผิดพลาด
- [ ] อัปเดตตัวอย่าง API ใน `req/req.txt` และ `README.md` (กรณีแก้ไข Endpoint)
- [ ] ไม่มีการเผลอใส่ไฟล์ `.env` หรือ Secret Key ลงใน Commit
