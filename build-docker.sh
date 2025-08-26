#!/bin/bash

echo "🏗️  Duck2api 本地Docker构建脚本"
echo "=================================="

# 设置变量
IMAGE_NAME="duck2api"
REGISTRY="ghcr.io/yaney01"
FULL_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}"

echo "📋 构建信息:"
echo "   镜像名称: ${IMAGE_NAME}"
echo "   注册表: ${REGISTRY}"
echo "   完整镜像: ${FULL_IMAGE_NAME}"
echo

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动Docker"
    exit 1
fi

echo "✅ Docker 环境检查通过"

# 构建镜像
echo "🔨 开始构建Docker镜像..."
if docker build -t ${IMAGE_NAME}:latest -t ${FULL_IMAGE_NAME}:latest .; then
    echo "✅ 镜像构建成功"
else
    echo "❌ 镜像构建失败"
    exit 1
fi

echo
echo "🎉 构建完成！"
echo
echo "💡 使用方法:"
echo "   本地运行:"
echo "   docker run -d --name duck2api -p 8080:8080 ${IMAGE_NAME}:latest"
echo
echo "   或使用完整镜像名:"
echo "   docker run -d --name duck2api -p 8080:8080 ${FULL_IMAGE_NAME}:latest"
echo

# 询问是否推送到GHCR
echo "🤔 是否要推送镜像到GitHub Container Registry? (需要先登录GHCR)"
echo "   如果选择是，请确保已经运行: docker login ghcr.io"
read -p "推送到GHCR? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🚀 推送镜像到GHCR..."
    if docker push ${FULL_IMAGE_NAME}:latest; then
        echo "✅ 镜像推送成功"
        echo "🌐 镜像现在可以通过以下命令使用:"
        echo "   docker run -d --name duck2api -p 8080:8080 ${FULL_IMAGE_NAME}:latest"
    else
        echo "❌ 镜像推送失败"
        echo "💡 请确保已经登录GHCR: docker login ghcr.io"
        echo "💡 用户名使用GitHub用户名，密码使用Personal Access Token"
    fi
else
    echo "⏭️  跳过推送，仅本地使用"
fi

echo
echo "🏁 脚本执行完成"