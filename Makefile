APP_NAME := moyu-assistant

# All available modules
ALL_MODULES := module_clock module_todo module_pomodoro module_notes module_jiggler

# Default: build with all modules (override with: make build MODULES="module_clock module_todo")
MODULES ?= $(ALL_MODULES)
TAGS := $(MODULES)

.PHONY: build build-all build-minimal run run-all clean help

## Build with selected modules (default: all)
build:
	go build -tags "$(TAGS)" -o $(APP_NAME) .

## Build for Windows with selected modules
build-win:
	GOOS=windows GOARCH=amd64 go build -tags "$(TAGS)" -ldflags "-H windowsgui" -o $(APP_NAME).exe .

## Build with all modules
build-all:
	go build -tags "$(ALL_MODULES)" -o $(APP_NAME) .

## Build with no modules (empty shell)
build-minimal:
	go build -o $(APP_NAME) .

## Run with selected modules
run:
	go run -tags "$(TAGS)" .

## Run with all modules
run-all:
	go run -tags "$(ALL_MODULES)" .

## Clean build artifacts
clean:
	rm -f $(APP_NAME) $(APP_NAME).exe

## Download dependencies
deps:
	go mod tidy

## Show help
help:
	@echo "摸鱼助手 - 构建帮助"
	@echo ""
	@echo "可用模块:"
	@echo "  module_clock     - 世界时钟"
	@echo "  module_todo      - 待办事项"
	@echo "  module_pomodoro  - 番茄钟"
	@echo "  module_notes     - 快捷笔记"
	@echo ""
	@echo "使用示例:"
	@echo "  make build                                        # 编译所有模块"
	@echo "  make build MODULES=\"module_clock module_todo\"      # 只编译时钟和待办"
	@echo "  make build-win                                     # 交叉编译 Windows 版本"
	@echo "  make build-minimal                                 # 不编译任何模块"
	@echo "  make run                                           # 运行（所有模块）"
	@echo "  make run MODULES=\"module_pomodoro\"                  # 只运行番茄钟模块"
