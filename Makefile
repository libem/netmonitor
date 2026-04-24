# Makefile for building and running the net-monitor application
.PHONY: build test run clean build-linux-amd64 build-windows-amd64 build-arm64

BIN_DIR := bin
APP := $(BIN_DIR)/net-monitor

# 默认构建
build:
	mkdir -p $(BIN_DIR)
	go build -o $(APP) ./cmd/monitor/main.go

# 运行测试
test:
	go test ./...

# 默认构建（Linux AMD64）
build-linux-amd64:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/net-monitor-linux ./cmd/monitor/main.go

# 构建（Windows AMD64）
build-windows-amd64:
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/net-monitor-windows ./cmd/monitor/main.go

# 默认构建网络监测工具（ARM）
build-linux-arm64:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BIN_DIR)/net-monitor-arm ./cmd/monitor/main.go

# 运行
run: build
	./$(APP)

# 清理构建产物
clean:
	go clean
	rm -rf $(BIN_DIR)/*
