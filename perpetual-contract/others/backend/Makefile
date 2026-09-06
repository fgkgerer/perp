# MetaNode Backend Makefile

.PHONY: all build run test clean gen api

# 默认目标
all: build

# 生成代码
gen:
	@echo "生成 API 代码..."
	goctl api go -api metanode.api -dir .

# 编译
build:
	@echo "编译项目..."
	go build -o bin/metanode metanode.go

# 运行
run:
	@echo "启动服务..."
	go run metanode.go -f etc/metanode.yaml

# 测试
test:
	@echo "运行测试..."
	go test -v ./...

# 清理
clean:
	@echo "清理..."
	rm -rf bin/
	go clean

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod tidy

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 检查代码
lint:
	@echo "检查代码..."
	golangci-lint run

# 生成数据库模型
model:
	@echo "生成数据库模型..."
	goctl model mysql ddl -src doc/sql/schema.sql -dir internal/model -c

# 帮助
help:
	@echo "可用命令:"
	@echo "  make gen     - 根据 API 定义生成代码"
	@echo "  make build   - 编译项目"
	@echo "  make run     - 启动服务"
	@echo "  make test    - 运行测试"
	@echo "  make clean   - 清理构建产物"
	@echo "  make deps    - 安装依赖"
	@echo "  make fmt     - 格式化代码"
	@echo "  make lint    - 检查代码"
	@echo "  make model   - 生成数据库模型"

