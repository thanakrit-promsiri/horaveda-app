#!/usr/bin/env bash

# ==============================================================================
# Script Name : test.sh
# Description : รัน Unit Tests ทั้งหมดในโปรเจกต์ และสร้างรายงาน Test Coverage
#               ทั้งแบบสรุปใน Terminal และแบบ HTML Report (coverage.html)
# ==============================================================================

# สั่งให้ Script หยุดทำงานทันทีหากมีคำสั่งใดคำสั่งหนึ่งส่งคืนค่า Error (Exit status != 0)
# เพื่อป้องกันการรันขั้นตอนต่อไปเมื่อ Unit Test ไม่ผ่าน
set -e

echo "=================================================================="
echo " 🚀 Starting Automated Testing & Test Coverage Process "
echo "=================================================================="

# ------------------------------------------------------------------------------
# Step 1: รัน Unit Tests ทั้งหมดในทุกแพ็กเกจย่อย (./...) แบบละเอียด (-v)
# ------------------------------------------------------------------------------
echo ""
echo "📌 Step 1: Running Unit Tests for all packages..."
go test -v ./...

# ------------------------------------------------------------------------------
# Step 2: คำนวณเปอร์เซ็นต์ Test Coverage รวมของแต่ละแพ็กเกจ
# ------------------------------------------------------------------------------
echo ""
echo "📌 Step 2: Calculating Test Coverage Summary per package..."
go test -cover ./...

# ------------------------------------------------------------------------------
# Step 3: ประมวลผลและส่งออกไฟล์ Profile ข้อมูล Coverage (coverage.out)
# ------------------------------------------------------------------------------
echo ""
echo "📌 Step 3: Generating Detailed Coverage Profile (coverage.out)..."
go test -coverprofile=coverage.out ./...

# ------------------------------------------------------------------------------
# Step 4: แสดงสถิติและเปอร์เซ็นต์ Coverage รายฟังก์ชันลงใน Terminal
# ------------------------------------------------------------------------------
echo ""
echo "📌 Step 4: Displaying Coverage Breakdown by Function:"
go tool cover -func=coverage.out

# ------------------------------------------------------------------------------
# Step 5: แปลงข้อมูลจาก coverage.out เป็นไฟล์รายงาน HTML (coverage.html)
# เพื่อใช้นำไปเปิดดูรายละเอียดการทดสอบบรรทัดต่อบรรทัดผ่าน Web Browser
# ------------------------------------------------------------------------------
echo ""
echo "📌 Step 5: Generating HTML Coverage Report (coverage.html)..."
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "=================================================================="
echo " ✅ All tests passed and coverage report created successfully!"
echo " 📄 Open 'coverage.html' in your browser to view line-by-line coverage."
echo "=================================================================="