# Duck2API（Yaney Fork）

将 DuckDuckGo AI Chat 转换为 OpenAI 兼容 API，支持 Chat Completions、Responses API、图像生成/编辑、文件问答、语音转文字、文字转语音、推理模式和网络搜索。

本仓库基于 [`aurora-develop/duck2api`](https://github.com/aurora-develop/duck2api) 维护，当前分支增加了图像候选筛选：当 Duck.ai 返回多张中间候选时，API 会自动选择解码后文件体积最大的有效图片，仅向客户端返回一张，避免 OpenWebUI 误取低质量预览图。

## 本 Fork 的改动

- 图像生成和图片编辑响应自动选择体积最大的 Base64 候选图。
- Docker 镜像发布到 `ghcr.io/yaney01/duck2api`。
- GitHub Actions 使用仓库自带的 `GITHUB_TOKEN` 构建并发布多架构镜像。
- 支持 `linux/amd64` 和 `linux/arm64`。

## 接口概览

| 功能 | 端点 | 说明 |
|---|---|---|
| Chat Completions | `POST /v1/chat/completions` | 流式/非流式对话 |
| Responses API | `POST /v1/responses` | OpenAI Responses API |
| 图像生成 | `POST /v1/images/generations` | 文生图，自动筛选最大候选 |
| 图像编辑 | `POST /v1/images/edits` | 图生图/改图，自动筛选最大候选 |
| 文件上传 | `POST /v1/files` | 上传文件用于问答 |
| 文件管理 | `GET/DELETE /v1/files/:id` | 查询、下载和删除文件 |
| 语音转文字 | `POST /v1/audio/transcriptions` | Whisper 兼容接口 |
| 文字转语音 | `POST /v1/audio/speech` | TTS 接口 |
| 模型列表 | `GET /v1/models` | 列出可用模型 |

完整 curl 示例见 [API.md](API.md)。

## Docker 部署

```bash
docker run -d \
  --name duck2api \
  --restart unless-stopped \
  -p 8080:8080 \
  ghcr.io/yaney01/duck2api:latest
```

验证服务：

```bash
curl http://127.0.0.1:8080/ping
curl http://127.0.0.1:8080/v1/models
```

更新镜像：

```bash
docker pull ghcr.io/yaney01/duck2api:latest
docker rm -f duck2api

docker run -d \
  --name duck2api \
  --restart unless-stopped \
  -p 8080:8080 \
  ghcr.io/yaney01/duck2api:latest
```

> GHCR 第一次构建后如果镜像仍是 Private，需要在 GitHub 仓库的 Packages 页面将 `duck2api` 包可见性改为 Public，或者先执行 `docker login ghcr.io` 后再拉取。

## Docker Compose 部署

```bash
mkdir -p duck2api && cd duck2api
curl -O https://raw.githubusercontent.com/yaney01/Duck2api/main/docker-compose.yml
docker compose up -d
```

更新：

```bash
docker compose pull
docker compose up -d
```

`docker-compose.yml` 默认使用：

```text
ghcr.io/yaney01/duck2api:latest
```

## 源码编译

要求 Go 版本与 `go.mod` 一致。

```bash
git clone https://github.com/yaney01/Duck2api.git
cd Duck2api
go mod download
go test ./...
go build -o duck2api .
chmod +x ./duck2api
./duck2api
```

默认监听：

```text
0.0.0.0:8080
```

修改端口：

```bash
SERVER_PORT=8081 ./duck2api
```

## OpenWebUI 配置

OpenWebUI 使用 Docker 部署时，API Base URL 填写：

```text
http://host.docker.internal:8080/v1
```

OpenWebUI 和 Duck2API 位于同一个 Docker Compose 网络时，可填写：

```text
http://duck2api:8080/v1
```

OpenWebUI 直接运行在本机时，可填写：

```text
http://127.0.0.1:8080/v1
```

没有设置 `Authorization` 环境变量时，OpenWebUI 的 API Key 字段可填写任意非空字符串，例如：

```text
duck2api
```

## 图像生成

```bash
curl http://127.0.0.1:8080/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "prompt": "一张高质量商业摄影作品",
    "n": 1,
    "reasoning_effort": "high",
    "response_format": "b64_json"
  }'
```

### 候选图筛选逻辑

Duck.ai 可能在一次请求中返回多张候选，即使请求参数是 `n: 1`。本 Fork 在序列化图像响应时执行以下逻辑：

1. 读取所有有效的 `b64_json` 候选。
2. 解码 Base64，并按真实图片字节数比较。
3. Base64 解码失败时，回退为比较字符串长度。
4. 仅返回体积最大的候选。
5. 如果上游只返回一张图，则保持原响应不变。

该规则用于过滤常见的低质量中间预览图。文件体积不能在所有情况下等同于视觉质量，但对当前 Duck.ai 返回的“第一张预览、第二张完整图”问题有效。

## 图像编辑

JSON Base64：

```bash
curl http://127.0.0.1:8080/v1/images/edits \
  -H "Content-Type: application/json" \
  -d '{
    "image": "<base64编码图片>",
    "prompt": "仅修改指定内容，保持其余部分不变",
    "model": "gpt-5.4-mini",
    "reasoning_effort": "high"
  }'
```

文件上传：

```bash
curl http://127.0.0.1:8080/v1/images/edits \
  -F "image=@input.png" \
  -F "prompt=仅修改指定内容，保持其余部分不变" \
  -F "model=gpt-5.4-mini"
```

## 支持的模型

以 `/v1/models` 实时返回为准。当前包含：

- `gpt-5.4-mini`
- `gpt-5.4-nano`
- `tinfoil/gpt-oss-120b`
- `claude-haiku-4-5`
- `mistral-small`

## 环境变量

| 变量 | 说明 | 示例 |
|---|---|---|
| `Authorization` | API 认证 Key | `Bearer your_key` |
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `8080` |
| `PROXY_URL` | 代理地址 | `http://proxy:8080` |
| `PREFIX` | URL 前缀 | `/api` |
| `TLS_CERT` | TLS 证书路径 | `/path/to/cert.pem` |
| `TLS_KEY` | TLS 私钥路径 | `/path/to/key.pem` |

## GitHub Actions 与镜像标签

推送到 `main` 后，`.github/workflows/build_docker.yml` 会先运行：

```bash
go test ./...
```

测试通过后构建并发布：

- `ghcr.io/yaney01/duck2api:latest`
- `ghcr.io/yaney01/duck2api:<VERSION>`
- `ghcr.io/yaney01/duck2api:v<VERSION>`
- `ghcr.io/yaney01/duck2api:<commit-sha前12位>`

## 上游与致谢

- 上游项目：[`aurora-develop/duck2api`](https://github.com/aurora-develop/duck2api)
- 参考项目：[`xqdoo00o/ChatGPT-to-API`](https://github.com/xqdoo00o/ChatGPT-to-API)

## License

MIT License
