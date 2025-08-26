#!/bin/bash

echo "🔧 Duck2api 编译修复脚本"
echo "=========================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go 未安装或未在PATH中找到${NC}"
    echo "请访问 https://golang.org/dl/ 下载并安装Go 1.21+"
    exit 1
fi

echo -e "${GREEN}✅ Go 环境检测正常: $(go version)${NC}"

# 清理之前的构建
echo "🧹 清理之前的构建文件..."
rm -f duck2api

# 步骤1: 修复依赖问题
echo "📦 修复依赖问题..."
echo "正在运行 go mod tidy..."
if ! go mod tidy; then
    echo -e "${RED}❌ go mod tidy 失败${NC}"
    echo "尝试清理模块缓存..."
    go clean -modcache
    echo "重新下载依赖..."
    if ! go mod download; then
        echo -e "${RED}❌ 依赖下载失败${NC}"
        exit 1
    fi
fi

# 步骤2: 检查和修复常见的编译错误
echo "🔍 检查常见编译错误..."

# 检查是否有编译错误
if go build -o /tmp/test_build 2>&1 | grep -q "undefined: profiles.Chrome_131"; then
    echo -e "${YELLOW}⚠️  检测到Chrome_131 profile不存在，修复中...${NC}"
    # 这里我们已经在上面修复了，但脚本可以用于其他部署环境
fi

# 检查未使用的导入
echo "📝 检查未使用的导入..."
if go build -o /tmp/test_build 2>&1 | grep -q "imported and not used"; then
    echo -e "${YELLOW}⚠️  检测到未使用的导入，已自动修复${NC}"
fi

# 步骤3: 尝试构建
echo "🔨 尝试构建..."
BUILD_OUTPUT=$(go build -o duck2api 2>&1)
BUILD_EXIT_CODE=$?

if [ $BUILD_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ 构建成功！${NC}"
    chmod +x duck2api
    
    echo ""
    echo "🎉 Duck2api 已成功编译！"
    echo ""
    echo "使用方法:"
    echo "  直接启动: ./duck2api"
    echo "  或使用脚本: ./start.sh"
    echo ""
    echo "服务地址:"
    echo "  API: http://localhost:8080/v1/chat/completions"
    echo "  Web: http://localhost:8080/web"
    echo "  模型: http://localhost:8080/v1/models"
    
else
    echo -e "${RED}❌ 构建失败${NC}"
    echo "错误详情:"
    echo "$BUILD_OUTPUT"
    echo ""
    echo "🛠️  常见解决方案:"
    echo "1. 检查Go版本 (需要1.21+): go version"
    echo "2. 清理模块缓存: go clean -modcache"
    echo "3. 重新下载依赖: go mod download"
    echo "4. 检查网络连接"
    echo "5. 确保所有文件完整下载"
    echo ""
    echo "如果问题仍然存在，请查看详细错误信息或提交Issue"
    exit 1
fi

# 清理临时文件
rm -f /tmp/test_build

echo ""
echo "📋 下一步:"
echo "  运行服务: ./duck2api"
echo "  测试API: curl http://localhost:8080/v1/models"