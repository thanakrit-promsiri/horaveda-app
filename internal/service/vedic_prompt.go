package service

import (
	"fmt"
	"strings"
)

// VedicAstrologerSystemPrompt System Prompt ประจำ Workspace บน AnythingLLM กำหนดบทบาทโหราจารย์พระเวท
const VedicAstrologerSystemPrompt = `คุณคือโหราจารย์ผู้เชี่ยวชาญโหราศาสตร์พระเวท (Vedic Jyotish / โหราศาสตร์ภารตะ) ระบบนิรายนะ อายนางศ์ Lahiri และระบบภพแบบ Whole Sign
- ตอบเป็นภาษาไทยที่สุภาพ ชัดเจน เข้าใจง่าย
- วิเคราะห์โดยอ้างอิงข้อมูลพื้นดวงที่ได้รับเท่านั้น ห้ามสร้างหรือสมมติตำแหน่งดาวที่ไม่มีอยู่ในข้อมูล
- ทุกข้อสรุปต้องระบุเหตุผลทางโหราศาสตร์ประกอบ (ดาว ราศี ภพ องศา หรือนวางศ์ที่เกี่ยวข้อง)
- ให้คำแนะนำเชิงบวกและสร้างสรรค์ หลีกเลี่ยงการทำนายที่ทำให้ตื่นตระหนก`

// vedicPredictionInstruction คำสั่งพยากรณ์พื้นดวงตามหลักโหราศาสตร์พระเวท (ประกอบกับพิกัดดวงดาว)
const vedicPredictionInstruction = `### คำอธิบายสัญลักษณ์ในข้อมูล
- ดาว: Su=อาทิตย์, Mo=จันทร์, Ma=อังคาร, Me=พุธ, Ju=พฤหัสบดี, Ve=ศุกร์, Sa=เสาร์, Ra=ราหู, Ke=เกตุ
- สถานะ: [R]=พักร์ (Retrograde), [C]=อัสตังคต (Combust), [Varg]=วรโคตตมะ (ราศีเดียวกันทั้ง D1 และ D9)
- ศักดิ์ราศี: (Exa)=อุจจ์, (Deb)=นิจ, (Own)=เกษตร, (Fri)=มิตรเกษตร, (Neu)=ราศีกลาง, (Ene)=ศัตรูเกษตร
- H1–H12 = ภพที่ 1–12 นับจากลัคนา, Nak = นักษัตร (บาท, ดาวเจ้านักษัตร), D9 = นวางศ์จักร
- Lords = เจ้าเรือนแต่ละภพและภพที่สถิต, Conj = ดาวที่อยู่ภพเดียวกัน, Dasha = วิมโศตรีทศา (MD=มหาทศา, AD=อันตรทศา)

### ขั้นตอนการวิเคราะห์ (ทำตามลำดับ)
1. **สภาพลัคนา (Lagna) ตนุลัคน์ และนักษัตร**
   - ราศีลัคนา องศา และดาวที่สถิตในภพที่ 1
   - ตนุลัคน์ (เจ้าเรือนลัคนา) สถิตภพใด ราศีใด ศักดิ์ราศีและความเข้มแข็งเป็นอย่างไร
   - นักษัตรของลัคนาและนักษัตรกำเนิดของจันทร์ (Janma Nakshatra) พร้อมบาทและดาวเจ้านักษัตร และผลต่ออุปนิสัย
2. **เกณฑ์สายตา (Drishti) และดาวกุมตามระยะองศา**
   - ใช้หลัก Graha Drishti: ทุกดาวมองภพที่ 7 จากตัวเอง, อังคารมองพิเศษภพที่ 4 และ 8, พฤหัสบดีมองพิเศษภพที่ 5 และ 9, เสาร์มองพิเศษภพที่ 3 และ 10, ราหู/เกตุมองภพที่ 5 และ 9
   - ระบุว่าดาวใดมองภพหรือดาวใด และเป็นอิทธิพลของศุภเคราะห์หรือบาปเคราะห์ โดยเฉพาะที่มองลัคนา ตนุลัคน์ และจันทร์
   - ดาวกุม: คำนวณระยะห่างองศาจริงของดาวที่อยู่ราศีเดียวกัน — ไม่เกิน 5° ถือว่ากุมแน่น (อิทธิพลสูง), 5°–10° กุมหลวม, เกิน 10° อิทธิพลอ่อน
   - พิจารณาดาวอัสตังคต [C] และดาวพักร์ [R] ประกอบด้วย
3. **ความเข้มแข็งของดาวผ่านนวางศ์จักร (D9)**
   - เปรียบเทียบศักดิ์ของดาวแต่ละดวงใน D1 กับ D9: วรโคตตมะ, อุจจ์/นิจใน D9, ดาวที่อ่อนใน D1 แต่แข็งใน D9 (หรือกลับกัน)
   - พิจารณาลัคนา D9 และตนุลัคน์ใน D9
   - สรุปดาวที่เข้มแข็งที่สุดและอ่อนแอที่สุดของดวงนี้
4. **สรุปคำพยากรณ์พื้นดวง**
   - อุปนิสัยและบุคลิกภาพ
   - การงาน การเงิน (ภพ 2, 10, 11 และเจ้าเรือน)
   - ความรักและคู่ครอง (ภพ 7, ศุกร์ และ D9)
   - สุขภาพ (ภพ 1, 6, 8)
   - ช่วงทศาปัจจุบัน (Now MD/AD) ส่งผลอย่างไร และคำแนะนำในการเสริมดวง

### รูปแบบคำตอบ
- ใช้หัวข้อ Markdown ตามขั้นตอนที่ 1–4
- อ้างอิงข้อมูลจากพื้นดวงประกอบทุกข้อ และห้ามสร้างตำแหน่งดาวที่ไม่มีในข้อมูล ไม่ต้องใช้สำนวนโหราศาสตร์โบราณเกินจำเป็น ให้ใช้คำอธิบายที่เข้าใจง่ายและชัดเจน`

// BuildVedicPredictionPrompt ประกอบพิกัดดวงดาวจาก Lahiri Prompt API เข้ากับคำสั่งพยากรณ์ตามหลักโหราศาสตร์พระเวท
func BuildVedicPredictionPrompt(chartPrompt string) string {
	var b strings.Builder
	b.WriteString("จงพยากรณ์พื้นดวงชะตากำเนิดตามหลักโหราศาสตร์พระเวท (Vedic Jyotish) จากพิกัดดวงดาวต่อไปนี้\n\n")
	b.WriteString("### ข้อมูลพื้นดวง (Sidereal Lahiri / Whole Sign)\n```\n")
	b.WriteString(strings.TrimSpace(chartPrompt))
	b.WriteString("\n```\n\n")
	b.WriteString(vedicPredictionInstruction)
	return b.String()
}

// thaiMonthNames รายชื่อเดือนภาษาไทย 1-12 (1-based index)
var thaiMonthNames = [...]string{
	"",
	"มกราคม",
	"กุมภาพันธ์",
	"มีนาคม",
	"เมษายน",
	"พฤษภาคม",
	"มิถุนายน",
	"กรกฎาคม",
	"สิงหาคม",
	"กันยายน",
	"ตุลาคม",
	"พฤศจิกายน",
	"ธันวาคม",
}

// ThaiMonthName แปลงลำดับเดือน (1-12) เป็นชื่อเดือนภาษาไทย
func ThaiMonthName(month int) string {
	if month >= 1 && month <= 12 {
		return thaiMonthNames[month]
	}
	return fmt.Sprintf("เดือนที่ %d", month)
}

// ToBuddhistYear แปลงปี ค.ศ. เป็น พ.ศ. (หากส่งปี พ.ศ. เกิน 2400 มาแล้วจะคงเดิมไว้)
func ToBuddhistYear(year int) int {
	if year > 0 && year < 2400 {
		return year + 543
	}
	return year
}

// FormatThaiMonthYear สร้างข้อความเดือนและปี พ.ศ. เช่น "เดือนมกราคม พ.ศ. 2570" หรือ "เดือนธันวาคม พ.ศ. 2569"
func FormatThaiMonthYear(year, month int) string {
	return fmt.Sprintf("เดือน%s พ.ศ. %d", ThaiMonthName(month), ToBuddhistYear(year))
}

// FormatThaiMonthYearShort สร้างข้อความเดือนและปีแบบย่อ เช่น "เดือนมกราคม 2570"
func FormatThaiMonthYearShort(year, month int) string {
	return fmt.Sprintf("เดือน%s %d", ThaiMonthName(month), ToBuddhistYear(year))
}

// BuildVedicTransitPredictionPrompt รวมข้อมูลดาวชะตากำเนิด (Natal Chart) และข้อมูลดาวจร (Transit Chart & Dasha)
// เข้ากับคำสั่งพยากรณ์ดาวจรประจำเดือนตามหลักโหราศาสตร์พระเวท โดยจัดรูปแบบคำตอบตามโครงสร้างที่กำหนดไว้อย่างเคร่งครัด
func BuildVedicTransitPredictionPrompt(natalPrompt, transitPrompt string, transitYear, transitMonth int) string {
	monthYearFull := FormatThaiMonthYear(transitYear, transitMonth)
	monthYearShort := FormatThaiMonthYearShort(transitYear, transitMonth)

	var b strings.Builder
	b.WriteString("จงพยากรณ์ดวงชะตาดาวจรประจำ")
	b.WriteString(monthYearFull)
	b.WriteString(" ตามหลักโหราศาสตร์พระเวท (Vedic Jyotish / โหราศาสตร์ภารตะ ระบบนิรายนะ Lahiri Ayanamsa และ Whole Sign) โดยวิเคราะห์ปฏิสัมพันธ์ระหว่างข้อมูลดาวชะตากำเนิด (Natal Chart) และข้อมูลดาวจรพร้อมวิมโศตรีทศา (Transit Chart & Dasha) ต่อไปนี้\n\n")

	b.WriteString("### 1. ข้อมูลดาวชะตากำเนิด (Natal Chart - Sidereal Lahiri / Whole Sign)\n```\n")
	b.WriteString(strings.TrimSpace(natalPrompt))
	b.WriteString("\n```\n\n")

	b.WriteString("### 2. ข้อมูลดาวจรและทศาประจำเดือน (Transit Chart & Dasha - Sidereal Lahiri / Whole Sign)\n```\n")
	b.WriteString(strings.TrimSpace(transitPrompt))
	b.WriteString("\n```\n\n")

	b.WriteString("### หลักเกณฑ์และขั้นตอนการวิเคราะห์\n")
	b.WriteString("1. ให้นำตำแหน่งดาวจร (Transit) และทศาปัจจุบัน (Vimshottari Dasha: Mahadasha, Antardasha, Pratyantardasha) มาพิจารณาสัมพันธ์กับ ลัคนา (Lagna), ตนุลัคน์ และจันทร์กำเนิด ของดาวชะตากำเนิด\n")
	b.WriteString("2. พิจารณาภพที่ดาวจรโคจรเข้าสถิต (nH..->tH..), การกุมกันของดาวจร (TransitConj) และเกณฑ์สายตาดาวจรส่งผลถึงดาวกำเนิด (TransitDrishti)\n")
	b.WriteString("3. นำเสนอคำทำนายเป็นภาษาไทยที่สุภาพ สร้างสรรค์ และมีเหตุผลทางโหราศาสตร์รองรับพอประมาณเน้นคำทำนาย\n\n")

	b.WriteString("### รูปแบบคำตอบที่ต้องส่งกลับ (ให้ใช้รูปแบบ โครงสร้าง และเครื่องหมายสัญลักษณ์ตามนี้อย่างเคร่งครัด):\n\n")
	b.WriteString("คำพยากรณ์ดวงชะตาประจำ")
	b.WriteString(monthYearFull)
	b.WriteString("\n\n")
	b.WriteString("อิทธิพลของดาวจรสำคัญใน")
	b.WriteString(monthYearFull)
	b.WriteString(" ส่งกระแสสัมพันธ์กับลัคนาตามเวลาเกิดของเจ้าชะตา มีเกณฑ์การเปลี่ยนแปลงไปในทิศทางที่โดดเด่นดังนี้ ภาพรวม อื่นๆ :\n")
	b.WriteString("[ระบุคำทำนายภาพรวมของเดือน อิทธิพลของดาวจรหลักและบรรยากาศทั่วไปในชีวิต]\n\n")
	b.WriteString("•\nการงานและการศึกษา: [คำทำนายด้านหน้าที่การงาน ความรับผิดชอบ โครงการใหม่ การสอบแข่งขัน หรือการติดต่อเจรจา]\n")
	b.WriteString("•\nการเงินและโชคลาภ: [คำทำนายด้านกระแสการเงิน รายรับ รายจ่าย การลงทุน หรือลาภผล]\n")
	b.WriteString("•\nความรักและสุขภาพ: [คำทำนายด้านความสัมพันธ์ คนรัก ครอบครัว และสุขภาพร่างกาย/สุขภาพจิตที่ควรดูแล]\n")
	b.WriteString("🪐 ดาวจรสำคัญ: [ระบุดาวเคราะห์จรที่มีบทบาทสำคัญและส่งกระแสสัมพันธ์เด่นชัด เช่น ดาวพฤหัสบดี ดาวศุกร์ ดาวเสาร์ หรือราหูจร ใน")
	b.WriteString(monthYearShort)
	b.WriteString("]\n")
	b.WriteString("✨ ")
	b.WriteString(monthYearShort)
	b.WriteString(": [คำแนะนำในการปรับปรุงตนเองจากการกระทำ (กรรม/สติตามหลักพุทธ) และการทำบุญในกรอบพระพุทธศาสนา เช่น ทำบุญค่าน้ำค่าไฟ ทำทาน บริจาคโลหิต ปฏิบัติธรรม เพื่อเสริมสิริมงคล]\n")

	return b.String()
}
