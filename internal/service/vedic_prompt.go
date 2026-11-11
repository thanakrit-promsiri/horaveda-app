package service

import (
	"fmt"
	"strings"
)

// VedicAstrologerSystemPrompt System Prompt ประจำ Workspace บน AnythingLLM กำหนดบทบาทโหราจารย์พระเวท
const VedicAstrologerSystemPrompt = `คุณคือโหราจารย์ผู้เชี่ยวชาญโหราศาสตร์พระเวท (Vedic Jyotish / ภารตะ) สายปาราศร (Parashara System) คำนวณตามระบบนิรายนะ (Sidereal Zodiac) อายนางศ์ Lahiri และระบบภพแบบ Whole Sign (1 ราศี = 1 ภพ)

แนวทางการวิเคราะห์และการตอบ:
1. ความเป็นกลางและตรรกะทางโหราศาสตร์:
   - ใช้หลักเหตุและผลตามกรรมวิบาก (Karma-vipāka) มองดวงชะตาเป็นแผนผังศักยภาพ บทเรียนชีวิต และจังหวะเวลา ไม่ทำนายในเชิงชะตากรรมกำหนดเด็ดขาด (No Fatalism)
   - แยกอิทธิพลของดาวเป็น 2 ระดับเสมอ: ศุภเคราะห์/บาปเคราะห์โดยกำเนิด (Naisargika) และศุภเคราะห์/บาปเคราะห์ตามหน้าที่ของลัคนา (Functional Benefic/Malefic/Yogakaraka)
   - วิเคราะห์โดยอ้างอิงข้อมูลพื้นดวงที่ได้รับเท่านั้น ห้ามคาดเดาหรือสมมุติตำแหน่งดาวที่ไม่มีในข้อมูลเด็ดขาด หากข้อมูลจุดใดไม่สมบูรณ์ให้ระบุตามจริง

2. ระดับภาษาและการสื่อสาร:
   - ตอบเป็นภาษาไทยที่กระชับ ชัดเจน เข้าใจง่าย ร่วมสมัย (Modern) หลีกเลี่ยงศัพท์แสงโบราณที่ตีความยาก แต่ยังคงความสุภาพ สุขุม หนักแน่น และน่าเชื่อถือในระดับมืออาชีพ
   - แปลงศัพท์เทคนิคโหราศาสตร์ภาษาสันสกฤตให้เป็นคำอธิบายที่อิงกับชีวิตจริง โดยกำกับศัพท์เทคนิคที่จำเป็นไว้ในวงเล็บเพียงเล็กน้อยเท่าที่จำเป็น
   - ให้คำแนะนำเชิงพัฒนาตนเอง (Self-development) และการปรับพฤติกรรม (Sattvic Upaya) หลีกเลี่ยงการทำนายเชิงลบที่สร้างความตระหนก`

// vedicPredictionInstruction คำสั่งพยากรณ์พื้นดวงตามหลักโหราศาสตร์พระเวท (ประกอบกับพิกัดดวงดาว)
const vedicPredictionInstruction = `### สัญลักษณ์และตัวย่อในข้อมูล
- ดาว (Graha): Su=อาทิตย์, Mo=จันทร์, Ma=อังคาร, Me=พุธ, Ju=พฤหัสบดี, Ve=ศุกร์, Sa=เสาร์, Ra=ราหู, Ke=เกตุ
- สภาพดาว: [R]=พักร์ (Retrograde), [C]=อัสตังคต (Combust), [Varg]=วรโคตตมะ (Vargottama - ราศีเดียวกันทั้ง D1 และ D9)
- ศักดิ์ดาว (Dignity): (Exa)=มหาอุจจ์, (Deb)=นิจ, (Own)=เกษตร/มูลตรีโกณ, (Fri)=มิตร, (Neu)=เป็นกลาง, (Ene)=ศัตรู
- ภพ (Bhava): H1–H12 (นับ H1 จากราศีลัคนา แบบ Whole Sign)
- โครงสร้าง: Nak=นักษัตรและบาท, D9=นวางศ์จักร, Lords=เจ้าเรือนภพ, Conj=ดาวกุมร่วมราศี, Dasha=วิมโศตรีทศา

---

### ขั้นตอนการวิเคราะห์เชิงตรรกะ (ให้ประมวลผลตามลำดับนี้)
1. การประเมินโครงสร้างหลัก: พิจารณาลัคนา (Lagna), ตนุลัคน์, จันทร์กำเนิด (Chandra Lagna) และจำแนกหน้าที่ดาว (Functional Benefic/Malefic/Yogakaraka) ประจำลัคนา
2. การตรวจสอบดาวและองศา: พิจารณาศักดิ์ดาว การกุม (พิจารณาระยะองศาชิดหรือห่าง) อาการอัสตังคต และเกณฑ์สายตา (Graha Drishti) โดยคำนึงถึงเกณฑ์พิเศษของดาวพฤหัสบดี(5,9) อังคาร(4,8) เสาร์(3,10)
3. ความเข้มแข็งเชิงลึก: ตรวจสอบความเสถียรของดาวผ่านนวางศ์จักร (D9) เช่น สภาพ [Varg] หรือการฟื้นตัวจากนิจ (Neecha Bhanga)
4. ทศาปัจจุบัน: วิเคราะห์ปฏิสัมพันธ์ระหว่างมหาทศา (MD) และอันตรทศา (AD) ตามหลักภพสัมพันธะ (Sambandha)

### รูปแบบคำตอบที่ต้องส่งกลับ (Output Format)
[คำสั่ง: จงสรุปผลการวิเคราะห์ทั้งหมดลงในโครงสร้างด้านล่างนี้อย่างเคร่งครัด ห้ามใช้ Format อื่น]

**1. แกนหลักของชีวิตและตัวตน**
[วิเคราะห์บุคลิกภาพ จุดแข็ง จุดอ่อน และสภาวะจิตใจ จากลัคนาและดาวจันทร์]

**2. ศักยภาพดวงชะตา**
- **การงานและความสำเร็จ:** [รูปแบบงานที่เหมาะสม อุปสรรค และความก้าวหน้า]
- **การเงินและทรัพย์สิน:** [แหล่งรายได้ สภาพคล่อง และศักยภาพการเก็บออม]
- **ความรักและคู่ครอง:** [ลักษณะคู่ครองและรูปแบบความสัมพันธ์]
- **สุขภาพและจุดพึงระวัง:** [ระบบร่างกายตามราศี/ดาวที่รับอิทธิพลบาปเคราะห์ เพื่อการดูแลเชิงป้องกัน]

**3. จังหวะชีวิตปัจจุบัน (วิมโศตรีทศา)**
[สรุปว่าอิทธิพลของทศา MD/AD ในปัจจุบันกำลังขับเคลื่อนชีวิตไปในทิศทางใด]

**4. แนวทางพัฒนาและปรับสมดุล (Sattvic Upaya)**
[คำแนะนำเชิงพฤติกรรมเพื่อลดจุดเสียและเสริมจุดแข็งตามเหตุผล หลีกเลี่ยงความเชื่อเชิงงมงาย]`

// BuildVedicPredictionPrompt ประกอบพิกัดดวงดาวเข้ากับคำสั่งพยากรณ์พื้นดวง
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
	"", "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
	"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
}

// ThaiMonthName แปลงลำดับเดือน (1-12) เป็นชื่อเดือนภาษาไทย
func ThaiMonthName(month int) string {
	if month >= 1 && month <= 12 {
		return thaiMonthNames[month]
	}
	return fmt.Sprintf("เดือนที่ %d", month)
}

// ToBuddhistYear แปลงปี ค.ศ. เป็น พ.ศ.
func ToBuddhistYear(year int) int {
	if year > 0 && year < 2400 {
		return year + 543
	}
	return year
}

// FormatThaiMonthYear สร้างข้อความเดือนและปี พ.ศ.
func FormatThaiMonthYear(year, month int) string {
	return fmt.Sprintf("เดือน%s พ.ศ. %d", ThaiMonthName(month), ToBuddhistYear(year))
}

// FormatThaiMonthYearShort สร้างข้อความเดือนและปีแบบย่อ
func FormatThaiMonthYearShort(year, month int) string {
	return fmt.Sprintf("เดือน%s %d", ThaiMonthName(month), ToBuddhistYear(year))
}

// BuildVedicTransitPredictionPrompt รวมข้อมูลชะตากำเนิดและดาวจร เพื่อพยากรณ์รายเดือน
func BuildVedicTransitPredictionPrompt(natalPrompt, transitPrompt string, transitYear, transitMonth int) string {
	monthYearFull := FormatThaiMonthYear(transitYear, transitMonth)
	monthYearShort := FormatThaiMonthYearShort(transitYear, transitMonth)

	var b strings.Builder
	b.WriteString("จงพยากรณ์ดวงดาวจรประจำ")
	b.WriteString(monthYearFull)
	b.WriteString(" ตามหลักโหราศาสตร์พระเวท โดยวิเคราะห์ข้อมูลดาวชะตากำเนิด (Natal) และดาวจร (Transit & Dasha) ดังนี้\n\n")

	b.WriteString("### 1. ข้อมูลดาวชะตากำเนิด (Natal Chart - Sidereal Lahiri / Whole Sign)\n```\n")
	b.WriteString(strings.TrimSpace(natalPrompt))
	b.WriteString("\n```\n\n")

	b.WriteString("### 2. ข้อมูลดาวจรและทศาประจำเดือน (Transit Chart & Dasha)\n```\n")
	b.WriteString(strings.TrimSpace(transitPrompt))
	b.WriteString("\n```\n\n")

	b.WriteString("### หลักเกณฑ์วิเคราะห์\n")
	b.WriteString("1. **ทศาคุมดาวจร:** ใช้ทศา (MD/AD) กำหนดศักยภาพหลัก และใช้ดาวจรเป็นตัวกระตุ้นจังหวะเวลา (Timing)\n")
	b.WriteString("2. **วิเคราะห์ 2 แกน:** เทียบ ลัคนา (เหตุการณ์ภายนอก) และ จันทร์กำเนิด (สภาวะจิตใจ)\n")
	b.WriteString("3. **ปฏิสัมพันธ์ดาวจร:** พิจารณาภพสถิต, องศากุม, เกณฑ์สายตาจร (Transit Drishti) และหน้าที่ดาวประจำลัคนา\n")
	b.WriteString("   - เน้นวิเคราะห์ **ดาวจรเร็ว** (อาทิตย์, จันทร์, อังคาร, พุธ, ศุกร์) เป็นหลัก\n")
	b.WriteString("   - ส่วน **ดาวจรช้า** (พฤหัสบดี, เสาร์, ราหู, เกตุ) ให้อ่านอิทธิพลภาพรวม\n\n")

	b.WriteString("### ข้อกำหนดการตอบ (Instruction)\n")
	b.WriteString("ให้คิดวิเคราะห์เชิงลึก (เช่น ทศา, นับภพขัดแย้ง 6/8, เกณฑ์สายตา) ภายในกระบวนการคิดของคุณ จากนั้น **สรุปเฉพาะคำทำนายตามรูปแบบด้านล่างนี้อย่างเคร่งครัด ห้ามแสดงวิธีคิดออกมา**\n\n")

	b.WriteString("### รูปแบบคำตอบที่ต้องส่งกลับ (Output Format):\n\n")
	b.WriteString("คำพยากรณ์ดวงชะตาประจำ")
	b.WriteString(monthYearFull)
	b.WriteString("\n\n")
	b.WriteString("อิทธิพลดาวจรสำคัญใน")
	b.WriteString(monthYearFull)
	b.WriteString(" ส่งผลต่อลัคนา มีทิศทางที่โดดเด่นดังนี้:\n")
	b.WriteString("[ระบุภาพรวมของเดือน เชื่อมโยงผลทศาปัจจุบันกับแรงกระตุ้นจากดาวจรหลัก]\n\n")
	b.WriteString("•\nการงานและการศึกษา: [ความรับผิดชอบ ทิศทางความก้าวหน้า การติดต่อเจรจา]\n")
	b.WriteString("•\nการเงินและโชคลาภ: [กระแสเงินสด สภาพคล่อง รายรับ รายจ่าย การลงทุน]\n")
	b.WriteString("•\nความรักและสุขภาพ: [ความสัมพันธ์คนรัก/ครอบครัว และสุขภาพกาย/ใจที่ควรใส่ใจ]\n")
	b.WriteString("🪐 ดาวจรสำคัญ: [ระบุดาวที่มีบทบาทหลัก (เน้นดาวจรเร็ว ส่วนดาวจรช้ากล่าวรวมๆ) พร้อมสรุปผลกระทบสั้นๆ ใน")
	b.WriteString(monthYearShort)
	b.WriteString("]\n")
	b.WriteString("✨ ")
	b.WriteString(monthYearShort)
	b.WriteString(": [คำแนะนำปรับพฤติกรรม (Sattvic Upaya) เพื่อรับมือดาวให้โทษ ใช้เหตุผลเชิงสร้างสรรค์ ไม่อ้างพิธีกรรม]\n")

	return b.String()
}
