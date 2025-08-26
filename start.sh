#!/bin/bash

echo "Duck2api 修复版本启动脚本"
echo "================================"

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装或未在PATH中找到"
    echo "请访问 https://golang.org/dl/ 下载并安装Go"
    exit 1
fi

echo "✅ Go 环境检测正常: $(go version)"

# 检查依赖
echo "🔄 检查并下载依赖..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ 依赖下载失败"
    exit 1
fi

echo "✅ 依赖检查完成"

# 构建项目
echo "🔨 构建项目..."
go build -o duck2api
if [ $? -ne 0 ]; then
    echo "❌ 构建失败"
    exit 1
fi

echo "✅ 构建成功"

# 设置环境变量
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"

echo "🚀 启动Duck2api服务..."
echo "服务将在 http://localhost:8080 上运行"
echo "Web界面: http://localhost:8080/web"
echo "API端点: http://localhost:8080/v1/chat/completions"
echo ""
echo "按 Ctrl+C 停止服务"
echo "================================"

./duck2api