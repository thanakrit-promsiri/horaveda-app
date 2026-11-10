package service

import (
	"fmt"
	"strings"
)

// VedicAstrologerSystemPrompt System Prompt ประจำ Workspace บน AnythingLLM
const VedicAstrologerSystemPrompt = `คุณคือโหราจารย์ Vedic Jyotish สาย Parashara ใช้ Sidereal/Lahiri และ Whole Sign (1 ราศี = 1 ภพ)

หลักตอบ:
- มองดวงเป็นศักยภาพ/บทเรียน/จังหวะชีวิต ไม่ fatalistic
- แยกศุภ/บาปโดยกำเนิดและตามหน้าที่ลัคนา (Functional Benefic/Malefic/Yogakaraka/Maraka/Badhaka)
- ใช้ข้อมูลที่ให้เท่านั้น ห้ามเดาหรือสมมุติตำแหน่งดาว; ข้อมูลไม่ครบให้ระบุ
- ตอบไทย สุภาพ หนักแน่น เข้าใจง่าย ใส่ศัพท์เทคนิคเท่าที่จำเป็น เน้นศัพท์ที่เหมาะกับวัยรุ่นและการใช้ชีวิตจริง
- ให้คำแนะนำพัฒนาตน/Upaya แบบ Sattvic หลีกเลี่ยงคำทำนายลบหรือสร้างความกลัว`

// vedicPredictionInstruction คำสั่งพยากรณ์พื้นดวงตามหลักโหราศาสตร์พระเวท
const vedicPredictionInstruction = `### สัญลักษณ์
Su=อาทิตย์ Mo=จันทร์ Ma=อังคาร Me=พุธ Ju=พฤหัสบดี Ve=ศุกร์ Sa=เสาร์ Ra=ราหู Ke=เกตุ
[R]=พักร์ [C]=อัสตังคต [Varg]=วรโคตตมะ
(Exa)=อุจจ์ (Deb)=นิจ (Own)=เกษตร (Fri)=มิตร (Neu)=กลาง (Ene)=ศัตรู
H1–H12=ภพ Whole Sign; Nak=นักษัตร/บาท; D9=นวางศ์; Lords=เจ้าเรือน; Conj=ร่วมราศี; Dasha=วิมโศตรีทศา (MD/AD)

### วิเคราะห์ตามลำดับ
1. โครงสร้างหลัก: ลัคนา (Lagna), ตนุลัคน์, องศา, นักษัตร, ดาว H1; จันทร์/นักษัตร/เจ้านักษัตร; จำแนก Yogakaraka, Functional Benefic/Malefic, Maraka/Badhaka
2. ตรวจ 9 ดาว: ภพ ศักดิ์ [R]/[C]; การกุม (Conj/Yuti) ตามองศา (≤5° สนิท, 5–10° ปานกลาง, >10° ร่วมภพ); Drishti 7th+พิเศษ (Ma 4/8, Ju 5/9, Sa 3/10, Ra/Ke 5/9); สรุปลัคน์/จันทร์/ภพสำคัญ
3. D9: เทียบศักดิ์ D1–D9; [Varg] เสถียร; อุจจ์/เกษตร D9=พลังซ่อน; นิจ D1 ดู Neecha Bhanga/ฟื้น D9; ลัคน์ D9=วาสนาครึ่งหลัง/คู่
4. สรุปรายด้าน: ตัวตน (H1/Su/Mo), เงิน (H2/H11/Ju), งาน (H10/เจ้าเรือน 10/Sa), รัก (H7/Ve/D9), สุขภาพ (H1/H6/H8) — เน้นจุดเด่น/จุดพัฒนา/คำแนะนำ
5. Dasha: วิเคราะห์ MD/AD (ภพ ศักดิ์ หน้าที่; สัมพันธะ 1/7, ตรีโกณ, 6/8, 2/12) + Upaya สัตตวะ: พฤติกรรม/ทาน/กุศลตามธาตุดาว; ไม่แนะนำอัญมณีแพง/ไสยศาสตร์/สะเดาะเคราะห์สร้างกลัว`

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

	b.WriteString("### หลักเกณฑ์และขั้นตอนการวิเคราะห์เชิงระบบ (Vedic Methodology)\n")
	b.WriteString("1. **ทศาคุมดาวจร (Dasha Primacy):** ใช้วิมโศตรีทศาปัจจุบัน (MD / AD / PD) เป็นแกนหลักในการกำหนดเรื่องราวและศักยภาพของช่วงเวลา จากนั้นใช้ตำแหน่งดาวจร (Transit/Gochara) เป็นตัวกระตุ้นจังหวะเวลา (Timing of Events)\n")
	b.WriteString("2. **วิเคราะห์ 2 มิติ (Dual Reference Frame):**\n")
	b.WriteString("   - เทียบกับลัคนา (Udaya Lagna): เพื่อประเมินเหตุการณ์ภายนอก สภาพแวดล้อมจริง และสถานะทางกายภาพ\n")
	b.WriteString("   - เทียบกับจันทร์กำเนิด (Chandra Lagna): เพื่อประเมินสภาวะอารมณ์ จิตใจ และการรับรู้ของเจ้าชะตา\n")
	b.WriteString("3. **ปฏิสัมพันธ์ของดาวเคราะห์ (Transit Aspects & Conjunctions):**\n")
	b.WriteString("   - พิจารณาภพที่ดาวจรโคจรเข้าสถิต (nH..->tH..) และศักดิ์ของดาวจรในราศีนั้นๆ\n")
	b.WriteString("   - พิจารณาดาวจรที่กุมดาวกำเนิดในองศาชิด และเกณฑ์สายตา (Graha Drishti) โดยเฉพาะเกณฑ์พิเศษของดาวครูและดาวใหญ่ (พฤหัสบดี เสาร์ อังคาร ราหู/เกตุ)\n")
	b.WriteString("   - ตระหนักถึงหน้าที่ของดาว (Functional Nature) ประจำลัคนานี้ ว่าดาวที่จรมาเป็นศุภเคราะห์หรือบาปเคราะห์ตามหน้าที่\n")
	b.WriteString("4. **ภาษาและการถ่ายทอด:** นำเสนอเป็นภาษาไทยที่สุภาพ หนักแน่น สุขุม เข้าใจง่าย และเป็นศัพท์ที่เหมาะสมกับวัยรุ่น  แปลงความหมายทางโหราศาสตร์ให้เป็นเหตุการณ์และแนวทางการใช้ชีวิตจริง หลีกเลี่ยงศัพท์เทคนิคที่ซับซ้อนเกินจำเป็น และไม่ทำนายในลักษณะชะตากรรมกำหนดเด็ดขาด (No Fatalism)\n\n")

	b.WriteString("### รูปแบบคำตอบที่ต้องส่งกลับ (ให้ใช้รูปแบบ โครงสร้าง และเครื่องหมายสัญลักษณ์ตามนี้อย่างเคร่งครัด):\n\n")
	b.WriteString("คำพยากรณ์ดวงชะตาประจำ")
	b.WriteString(monthYearFull)
	b.WriteString("\n\n")
	b.WriteString("อิทธิพลของดาวจรสำคัญใน")
	b.WriteString(monthYearFull)
	b.WriteString(" ส่งกระแสสัมพันธ์กับลัคนาตามเวลาเกิดของเจ้าชะตา มีเกณฑ์การเปลี่ยนแปลงไปในทิศทางที่โดดเด่นดังนี้ ภาพรวม อื่นๆ :\n")
	b.WriteString("[ระบุคำทำนายภาพรวมของเดือน บรรยากาศหลักในชีวิต โดยเชื่อมโยงระหว่างผลของทศาปัจจุบันกับแรงกระตุ้นจากดาวจรหลัก]\n\n")
	b.WriteString("•\nการงานและการศึกษา: [คำทำนายด้านหน้าที่การงาน ความรับผิดชอบ โครงการ ทิศทางความก้าวหน้า การสอบแข่งขัน หรือการติดต่อเจรจา]\n")
	b.WriteString("•\nการเงินและโชคลาภ: [คำทำนายด้านกระแสเงินสด สภาพคล่อง รายรับ รายจ่าย การลงทุน หรือลาภผลตามจังหวะดาว]\n")
	b.WriteString("•\nความรักและสุขภาพ: [คำทำนายด้านความสัมพันธ์กับคนรัก ครอบครัว และสภาพร่างกาย/จิตใจที่ควรใส่ใจดูแลเป็นพิเศษ]\n")
	b.WriteString("🪐 ดาวจรสำคัญ: [ระบุดาวเคราะห์จรที่มีบทบาทหลักในการส่งกระแสสัมพันธ์ เช่น พฤหัสบดี เสาร์ ราหู เกตุ หรือดาวประจำทศา พร้อมสรุปจุดส่งผลกระทบสั้นๆ ใน")
	b.WriteString(monthYearShort)
	b.WriteString("]\n")
	b.WriteString("✨ ")
	b.WriteString(monthYearShort)
	b.WriteString(": [คำแนะนำในการดำเนินชีวิตเชิงสติและปัญญา การปรับพฤติกรรมและการกระทำ (กุศลกรรม) ให้สอดคล้องกับสภาวะดาว รวมถึงการสละออก การเกื้อกูลสังคม หรือการทำทานที่สอดคล้องกับธาตุของดาวที่ให้โทษเพื่อปรับสมดุล โดยใช้เหตุผลเชิงสร้างสรรค์ ไม่กล่าวอ้างชื่อศาสนาหรือความเชื่อเชิงพิธีกรรม]\n")

	return b.String()
}
