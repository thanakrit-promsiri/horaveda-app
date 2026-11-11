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

// BuildVedicTransitPredictionPrompt รวมข้อมูลดาวชะตากำเนิดและดาวจร เพื่อพยากรณ์รายเดือน
func BuildVedicTransitPredictionPrompt(natalPrompt, transitPrompt string, transitYear, transitMonth int) string {
	monthYearFull := FormatThaiMonthYear(transitYear, transitMonth)
	monthYearShort := FormatThaiMonthYearShort(transitYear, transitMonth)

	var b strings.Builder
	b.WriteString("จงพยากรณ์ดวงชะตาดาวจรประจำ")
	b.WriteString(monthYearFull)
	b.WriteString(" ตามหลักโหราศาสตร์พระเวท โดยวิเคราะห์ปฏิสัมพันธ์ระหว่างข้อมูลดาวชะตากำเนิด (Natal) และดาวจร (Transit & Dasha) ต่อไปนี้\n\n")

	b.WriteString("### 1. ข้อมูลดาวชะตากำเนิด (Natal Chart - Sidereal Lahiri / Whole Sign)\n```\n")
	b.WriteString(strings.TrimSpace(natalPrompt))
	b.WriteString("\n```\n\n")

	b.WriteString("### 2. ข้อมูลดาวจรและทศาประจำเดือน (Transit Chart & Dasha)\n```\n")
	b.WriteString(strings.TrimSpace(transitPrompt))
	b.WriteString("\n```\n\n")

	b.WriteString("### หลักเกณฑ์การวิเคราะห์เชิงระบบ (ให้ประมวลผลตามตรรกะนี้)\n")
	b.WriteString("1. **ทศาคุมดาวจร:** ใช้วิมโศตรีทศาปัจจุบัน (MD / AD) กำหนดศักยภาพหลัก และใช้ดาวจรเป็นตัวกระตุ้นจังหวะเวลา (Timing)\n")
	b.WriteString("2. **วิเคราะห์ 2 แกน (Dual Reference Frame):** ประเมินร่วมกันระหว่าง ลัคนา (เหตุการณ์ภายนอก) และ จันทร์กำเนิด (สภาวะจิตใจ)\n")
	b.WriteString("3. **ปฏิสัมพันธ์ดาวจร:** พิจารณาภพที่ดาวโคจรเข้าสถิต การกุมองศาชิดดาวกำเนิด และเกณฑ์สายตาจร (Transit Drishti) โดยตระหนักถึงหน้าที่ดาว (Functional Nature) ประจำลัคนาดวงนี้เสมอ\n\n")

	// เพิ่มคำสั่ง CoT (Chain of Thought)
	b.WriteString("### ขั้นตอนการสร้างคำตอบ (Instruction)\n")
	b.WriteString("คุณสามารถวิเคราะห์ความสัมพันธ์เชิงลึก (เช่น การนับภพขัดแย้ง 6/8, การส่งเกณฑ์สายตา, อิทธิพลทศา) ในกระบวนการคิดของคุณ จากนั้น **ให้สังเคราะห์ผลลัพธ์เพื่อนำเสนอเฉพาะคำทำนายตามโครงสร้างและสัญลักษณ์ด้านล่างนี้อย่างเคร่งครัด โดยไม่ต้องแสดงวิธีคิดออกมา**\n\n")

	b.WriteString("### รูปแบบคำตอบที่ต้องส่งกลับ (Output Format):\n\n")
	b.WriteString("คำพยากรณ์ดวงชะตาประจำ")
	b.WriteString(monthYearFull)
	b.WriteString("\n\n")
	b.WriteString("อิทธิพลของดาวจรสำคัญใน")
	b.WriteString(monthYearFull)
	b.WriteString(" ส่งกระแสสัมพันธ์กับลัคนาตามเวลาเกิดของเจ้าชะตา มีเกณฑ์การเปลี่ยนแปลงไปในทิศทางที่โดดเด่นดังนี้ ภาพรวม อื่นๆ :\n")
	b.WriteString("[ระบุคำทำนายภาพรวมของเดือน บรรยากาศหลักในชีวิต โดยเชื่อมโยงระหว่างผลของทศาปัจจุบันกับแรงกระตุ้นจากดาวจรหลัก]\n\n")
	b.WriteString("•\nการงานและการศึกษา: [คำทำนายด้านหน้าที่การงาน ความรับผิดชอบ ทิศทางความก้าวหน้า หรือการติดต่อเจรจา]\n")
	b.WriteString("•\nการเงินและโชคลาภ: [คำทำนายด้านกระแสเงินสด สภาพคล่อง รายรับ รายจ่าย การลงทุน]\n")
	b.WriteString("•\nความรักและสุขภาพ: [คำทำนายด้านความสัมพันธ์กับคนรัก ครอบครัว และสภาพร่างกาย/จิตใจที่ควรใส่ใจ]\n")
	b.WriteString("🪐 ดาวจรสำคัญ: [ระบุดาวเคราะห์จรที่มีบทบาทหลักในการส่งกระแสสัมพันธ์ เช่น พฤหัสบดี เสาร์ ราหู เกตุ พร้อมสรุปจุดส่งผลกระทบสั้นๆ ใน")
	b.WriteString(monthYearShort)
	b.WriteString("]\n")
	b.WriteString("✨ ")
	b.WriteString(monthYearShort)
	b.WriteString(": [คำแนะนำในการดำเนินชีวิตเชิงสติและปัญญา การปรับพฤติกรรม (Sattvic Upaya) เพื่อปรับสมดุลตามดาวที่ให้โทษ โดยใช้เหตุผลเชิงสร้างสรรค์ ไม่กล่าวอ้างความเชื่อเชิงพิธีกรรม]\n")

	return b.String()
}
