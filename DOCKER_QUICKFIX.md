# 🚨 Docker镜像暂不可用 - 快速解决方案

## 问题说明
GitHub Container Registry (GHCR) 中的 `ghcr.io/yaney01/duck2api:latest` 镜像尚未构建完成。

## 🛠️ 立即可用的解决方案

### 方案1: 本地构建（推荐）

```bash
# 1. 克隆仓库
git clone https://github.com/yaney01/Duck2api.git
cd Duck2api

# 2. 使用一键构建脚本
./build-docker.sh

# 3. 运行容器
docker run -d --name duck2api -p 8080:8080 duck2api:latest
```

### 方案2: 手动构建

```bash
# 1. 克隆仓库
git clone https://github.com/yaney01/Duck2api.git
cd Duck2api

# 2. 构建镜像
docker build -t duck2api .

# 3. 运行容器
docker run -d --name duck2api -p 8080:8080 duck2api

# 4. 验证服务
curl http://localhost:8080/v1/models
```

### 方案3: 直接运行源码

```bash
# 1. 克隆仓库
git clone https://github.com/yaney01/Duck2api.git
cd Duck2api

# 2. 使用启动脚本（需要Go环境）
./start.sh

# 或手动运行
go build -o duck2api
./duck2api
```

## 🔄 自动化解决方案

GitHub Actions正在配置中，将自动构建镜像。你也可以：

1. **检查构建状态**: https://github.com/yaney01/Duck2api/actions
2. **手动触发构建**: 在Actions页面点击"Run workflow"
3. **等待自动构建**: 下次推送代码时会自动构建

## 📋 验证服务运行

无论使用哪种方案，服务启动后可以通过以下方式验证：

```bash
# 检查容器状态
docker ps

# 测试API
curl http://localhost:8080/v1/models

# 访问Web界面
open http://localhost:8080/web
```

## 🎯 预期输出

成功运行后，你应该看到：

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o-mini",
      "object": "model",
      "created": 1685474247,
      "owned_by": "duckduckgo"
    },
    ...
  ]
}
```

## ⚡ 快速测试

```bash
# 测试聊天API
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

---

**注意**: 这个问题是临时的，GitHub Actions配置完成后，`ghcr.io/yaney01/duck2api:latest` 将可以正常使用。