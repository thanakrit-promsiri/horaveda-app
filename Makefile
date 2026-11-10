.PHONY: all build run test coverage clean tidy vet

BINARY_NAME=bin/server
MAIN_PATH=cmd/server/main.go

all: test build

## build: คอมไพล์ไบนารีของเซิร์ฟเวอร์ไว้ที่ bin/server
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

## run: รันเซิร์ฟเวอร์ในโหมดพัฒนา
run:
	go run $(MAIN_PATH)

## test: รัน Unit Tests ทั้งหมดทุกแพ็กเกจ
test:
	go test -v -count=1 ./...

## coverage: รัน Unit Tests พร้อมสร้างรายงาน Coverage (coverage.out และ coverage.html)
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## vet: ตรวจสอบความถูกต้องของโค้ดด้วย go vet
vet:
	go vet ./...

## tidy: จัดระเบียบ go.mod และ go.sum
tidy:
	go mod tidy

## clean: ลบไฟล์ไบนารีและไฟล์รายงานผลการทดสอบ
clean:
	rm -rf bin/ server.exe coverage.out coverage.html
