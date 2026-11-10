#!/usr/bin/env bash

# ==============================================================================
# Script Name : run.sh
# Description : รัน Unit Tests, คำนวณ Test Coverage, ทำการ Build Binary
#               และเริ่มต้นเปิดใช้งานเซิร์ฟเวอร์ (Horaveda API Server)
# ==============================================================================

# สั่งให้หยุดทำงานทันทีหากมีคำสั่งใดเกิดข้อผิดพลาด (Exit code != 0)
set -e

# กำหนดตัวแปรและแฟล็กเริ่มต้น
SKIP_TEST=false
NO_RUN=false
SERVER_ARGS=()

# รองรับพารามิเตอร์เสริม เช่น --skip-test หรือ --no-run
for arg in "$@"; do
    case "$arg" in
        --skip-test|-s)
            SKIP_TEST=true
            ;;
        --no-run|-b|--build-only)
            NO_RUN=true
            ;;
        *)
            SERVER_ARGS+=("$arg")
            ;;
    esac
done

echo "=================================================================="
echo " 🚀 Horaveda App: Test Coverage, Build & Run Script"
echo "=================================================================="

# ------------------------------------------------------------------------------
# ขั้นตอนที่ 1: Unit Test & Coverage Profile
# ------------------------------------------------------------------------------
if [[ "$SKIP_TEST" == "true" ]]; then
    echo ""
    echo "⏩ ข้ามขั้นตอน Unit Test & Coverage ตามคำสั่ง (--skip-test)"
else
    echo ""
    echo "🧪 ขั้นตอนที่ 1: กำลังรัน Unit Tests และเก็บข้อมูล Test Coverage..."
    go test -v -coverprofile=coverage.out -covermode=atomic ./...

    echo ""
    echo "📊 สรุปเปอร์เซ็นต์ Test Coverage รายฟังก์ชัน:"
    go tool cover -func=coverage.out

    echo ""
    echo "📄 กำลังสร้างรายงาน HTML Coverage (coverage.html)..."
    go tool cover -html=coverage.out -o coverage.html
    echo "✅ บันทึกรายงาน Test Coverage ที่ 'coverage.out' และ 'coverage.html' เรียบร้อยแล้ว"
fi

# ------------------------------------------------------------------------------
# ขั้นตอนที่ 2: Build Production Binary
# ------------------------------------------------------------------------------
echo ""
echo "🔨 ขั้นตอนที่ 2: กำลังทำการ Build Binary (bin/server)..."
mkdir -p bin
go build -v -o bin/server cmd/server/main.go
echo "✅ ทำการ Build Binary สำเร็จแล้ว"

# ------------------------------------------------------------------------------
# ขั้นตอนที่ 3: Start Server
# ------------------------------------------------------------------------------
if [[ "$NO_RUN" == "true" ]]; then
    echo ""
    echo "🏁 เสร็จสิ้นขั้นตอน Test และ Build เรียบร้อยแล้ว (--no-run)"
    echo "=================================================================="
    exit 0
fi

echo ""
echo "🌟 ขั้นตอนที่ 3: เริ่มต้นรันเซิร์ฟเวอร์ Horaveda App..."
echo "=================================================================="

# ตรวจสอบไฟล์ Binary สำหรับรันตามระบบปฏิบัติการ
if [[ -f "./bin/server" ]]; then
    exec ./bin/server "${SERVER_ARGS[@]}"
elif [[ -f "./bin/server.exe" ]]; then
    exec ./bin/server.exe "${SERVER_ARGS[@]}"
else
    exec go run cmd/server/main.go "${SERVER_ARGS[@]}"
fi
