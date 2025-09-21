PROJECT_NAME := blockcooker

SRC_DIR := .
MAIN_FILE := cmd/main.go

BUILD_DIR := ./bin


all: build-linux-arm build-linux-x86 build-darwin-arm build-windows-x86

build-linux-arm:
	@echo "--- 正在编译 ARM 架构可执行文件 ---"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(PROJECT_NAME)-linux-arm64 $(SRC_DIR)/$(MAIN_FILE)
	@echo "编译完成：$(BUILD_DIR)/$(PROJECT_NAME)-arm64"

build-darwin-arm:
	@echo "--- 正在编译 MacOS ARM 架构可执行文件 ---"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(PROJECT_NAME)-darwin-arm64 $(SRC_DIR)/$(MAIN_FILE)
	@echo "编译完成：$(BUILD_DIR)/$(PROJECT_NAME)-arm64"

build-linux-x86:
	@echo "--- 正在编译 x86 架构可执行文件 ---"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(PROJECT_NAME)-linux-amd64 $(SRC_DIR)/$(MAIN_FILE)
	@echo "编译完成：$(BUILD_DIR)/$(PROJECT_NAME)-amd64"

build-windows-x86:
	@echo "--- 正在编译 Windows x86 架构可执行文件 ---"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(PROJECT_NAME)-windows-amd64.exe $(SRC_DIR)/$(MAIN_FILE)
	@echo "编译完成：$(BUILD_DIR)/$(PROJECT_NAME)-windows-amd64.exe"
