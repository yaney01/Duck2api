#!/bin/bash

echo "Duck2api 修复版本启动脚本"
echo "================================"

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装或未在PATH中找到"
    echo "请访问 https://golang.org/dl/ 下载并安装Go 1.21+"
    exit 1
fi

echo "✅ Go 环境检测正常: $(go version)"

# 检查Go版本
GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
REQUIRED_VERSION="1.21"
if [[ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]]; then
    echo "⚠️  警告: 检测到Go版本为 $GO_VERSION，建议使用Go 1.21+"
fi

# 检查依赖
echo "🔄 检查并下载依赖..."
if ! go mod tidy; then
    echo "❌ 依赖下载失败"
    echo "💡 尝试清理模块缓存: go clean -modcache"
    exit 1
fi

echo "✅ 依赖检查完成"

# 清理之前的构建
if [ -f "duck2api" ]; then
    echo "🧹 清理之前的构建文件..."
    rm -f duck2api
fi

# 构建项目
echo "🔨 构建项目..."
if ! go build -o duck2api; then
    echo "❌ 构建失败"
    echo "💡 常见解决方案:"
    echo "   1. 检查Go版本: go version (需要1.21+)"
    echo "   2. 清理依赖: go mod tidy"
    echo "   3. 清理缓存: go clean -modcache && go mod download"
    echo "   4. 检查网络连接"
    exit 1
fi

# 检查构建产物
if [ ! -f "duck2api" ]; then
    echo "❌ 构建文件不存在"
    exit 1
fi

echo "✅ 构建成功"

# 设置执行权限
chmod +x duck2api

# 设置环境变量
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"

# 检查端口占用
if command -v netstat &> /dev/null; then
    if netstat -tlnp 2>/dev/null | grep -q ":8080 "; then
        echo "⚠️  警告: 端口8080已被占用"
        echo "💡 请停止占用端口的进程或修改SERVER_PORT环境变量"
    fi
fi

echo "🚀 启动Duck2api服务..."
echo "服务将在 http://localhost:8080 上运行"
echo "Web界面: http://localhost:8080/web"
echo "API端点: http://localhost:8080/v1/chat/completions"
echo "模型列表: http://localhost:8080/v1/models"
echo ""
echo "按 Ctrl+C 停止服务"
echo "================================"
echo ""

# 启动服务
./duck2api