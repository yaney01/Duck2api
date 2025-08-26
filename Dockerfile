# 使用 Go 1.21 官方镜像作为构建环境
FROM golang:1.21 AS builder

# 禁用 CGO
ENV CGO_ENABLED=0

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码并构建应用
COPY . .
RUN go build -ldflags "-s -w" -o /app/duck2api .

# 使用 Alpine Linux 作为最终镜像
FROM alpine:latest

# 添加标签信息
LABEL maintainer="yaney01" \
      description="Duck2api - DuckDuckGo AI Chat API Proxy (Fixed Version)" \
      version="1.1.0-fix" \
      repository="https://github.com/yaney01/Duck2api"

# 安装必要的包
RUN apk --no-cache add ca-certificates curl tzdata

# 设置时区
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone

# 创建非root用户
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的应用
COPY --from=builder /app/duck2api /app/duck2api

# 更改文件所有权
RUN chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD curl -f http://localhost:8080/v1/models || exit 1

# 运行应用
CMD ["/app/duck2api"]
