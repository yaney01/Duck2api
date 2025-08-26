# Duck2api Docker 部署指南

## 🐳 快速开始

### 使用预构建镜像（推荐）

```bash
# 拉取并运行最新的修复版本
docker run -d \
  --name duck2api \
  -p 8080:8080 \
  ghcr.io/yaney01/duck2api:latest
```

访问服务：
- API: `http://localhost:8080/v1/chat/completions`
- Web界面: `http://localhost:8080/web`
- 模型列表: `http://localhost:8080/v1/models`

### 使用Docker Compose（推荐生产环境）

1. 创建项目目录：
```bash
mkdir duck2api && cd duck2api
```

2. 下载docker-compose.yml：
```bash
curl -O https://raw.githubusercontent.com/yaney01/Duck2api/main/docker-compose.yml
```

3. 启动服务：
```bash
docker-compose up -d
```

4. 查看日志：
```bash
docker-compose logs -f duck2api
```

## 📋 环境变量配置

### 基本配置
```bash
# 服务器配置
SERVER_HOST=0.0.0.0          # 监听地址
SERVER_PORT=8080             # 监听端口
```

### 可选配置
```bash
# 认证配置
Authorization=your_auth_key  # API认证密钥

# 代理配置
PROXY_URL=http://proxy:port  # 上游代理

# TLS配置
TLS_CERT=/path/to/cert.pem   # TLS证书路径
TLS_KEY=/path/to/key.pem     # TLS私钥路径
```

### 使用环境变量文件

创建 `.env` 文件：
```env
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
# Authorization=your_secret_key
# PROXY_URL=http://your-proxy:8080
```

使用环境变量文件启动：
```bash
docker run -d \
  --name duck2api \
  -p 8080:8080 \
  --env-file .env \
  ghcr.io/yaney01/duck2api:latest
```

## 🏗️ 本地构建

如果你想自己构建镜像：

```bash
# 克隆仓库
git clone https://github.com/yaney01/Duck2api.git
cd Duck2api

# 构建镜像
docker build -t duck2api:local .

# 运行本地构建的镜像
docker run -d \
  --name duck2api \
  -p 8080:8080 \
  duck2api:local
```

## 🔧 高级配置

### 使用自定义配置

创建完整的docker-compose.yml配置：

```yaml
version: '3.8'

services:
  duck2api:
    image: ghcr.io/yaney01/duck2api:latest
    container_name: duck2api
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      # 取消注释以启用认证
      # - Authorization=your-secret-api-key
      # 取消注释以使用代理
      # - PROXY_URL=http://your-proxy:8080
    volumes:
      # 挂载配置文件（可选）
      - ./config:/app/config
      # 挂载日志目录（可选）
      - ./logs:/app/logs
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/v1/models"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - duck2api_network

  # 可选：添加Nginx反向代理
  nginx:
    image: nginx:alpine
    container_name: duck2api-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl/certs
    depends_on:
      - duck2api
    networks:
      - duck2api_network

networks:
  duck2api_network:
    driver: bridge
```

### 使用反向代理

创建 `nginx.conf` 配置：

```nginx
events {
    worker_connections 1024;
}

http {
    upstream duck2api {
        server duck2api:8080;
    }

    server {
        listen 80;
        server_name your-domain.com;

        location / {
            proxy_pass http://duck2api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            
            # 支持流式响应
            proxy_buffering off;
            proxy_cache off;
        }
    }
}
```

## 📊 监控和日志

### 查看容器状态
```bash
# 查看运行状态
docker ps

# 查看资源使用
docker stats duck2api

# 查看健康检查
docker inspect duck2api | grep -A 10 '"Health"'
```

### 查看日志
```bash
# 查看最新日志
docker logs duck2api

# 实时跟踪日志
docker logs -f duck2api

# 查看最近100行日志
docker logs --tail 100 duck2api
```

### 性能监控

可以使用Prometheus + Grafana监控：

```yaml
# 添加到docker-compose.yml
  prometheus:
    image: prom/prometheus
    container_name: duck2api-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    container_name: duck2api-grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## 🐛 故障排除

### 常见问题

1. **容器启动失败**
   ```bash
   # 检查日志
   docker logs duck2api
   
   # 检查端口占用
   netstat -tlnp | grep 8080
   ```

2. **无法访问服务**
   ```bash
   # 检查防火墙
   sudo ufw status
   
   # 检查容器网络
   docker network ls
   docker inspect duck2api
   ```

3. **403/429错误频繁**
   ```bash
   # 使用代理
   docker run -d \
     --name duck2api \
     -p 8080:8080 \
     -e PROXY_URL=http://your-proxy:port \
     ghcr.io/yaney01/duck2api:latest
   ```

### 重置和清理

```bash
# 停止并删除容器
docker stop duck2api
docker rm duck2api

# 删除镜像
docker rmi ghcr.io/yaney01/duck2api:latest

# 清理未使用的资源
docker system prune -a
```

## 🔄 更新和维护

### 更新到最新版本

```bash
# 拉取最新镜像
docker pull ghcr.io/yaney01/duck2api:latest

# 重启容器
docker-compose down
docker-compose up -d
```

### 自动更新

可以使用Watchtower自动更新：

```yaml
# 添加到docker-compose.yml
  watchtower:
    image: containrrr/watchtower
    container_name: duck2api-watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_INCLUDE_RESTARTING=true
    command: duck2api
```

## 📞 支持

如果遇到问题：
1. 查看 [FIXES.md](../FIXES.md) 了解修复详情
2. 在 [GitHub Issues](https://github.com/yaney01/Duck2api/issues) 提交问题
3. 检查 [GitHub Actions](https://github.com/yaney01/Duck2api/actions) 构建状态