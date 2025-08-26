# Duck2api 修复版本说明

## 问题诊断

Duck2api项目不能使用的主要原因包括：

1. **DuckDuckGo加强了反机器人检测**
2. **TLS指纹检测更加严格**
3. **用户代理识别机制升级**
4. **Token获取机制变化**
5. **请求频率限制加强**

## 修复内容

### 1. 升级TLS指纹模拟
- 将TLS客户端配置从 `Okhttp4Android13` 升级到 `Chrome_131`
- 添加 `WithInsecureSkipVerify()` 选项
- 缩短超时时间从600秒到120秒，提高响应性

### 2. 改进用户代理策略
- 更新Chrome版本从120到131
- 添加用户代理随机化功能，包含多个真实的浏览器版本
- 添加随机延迟以模拟人类行为

### 3. 增强请求头
- 更新 `sec-ch-ua` 头部信息
- 添加 `accept-encoding` 支持
- 添加 `cache-control` 和 `pragma` 头部
- 将语言设置从中文改为英文，减少检测风险

### 4. 改进Token管理
- 添加重试机制，最多尝试3次获取token
- 实现递增延迟策略
- 缩短token有效期从3分钟到2分钟
- 更好的错误处理和日志记录

### 5. 增强错误处理
- 针对403、429、502等状态码提供详细错误信息
- 添加重试建议
- 改进错误消息的可读性

### 6. 添加请求重试机制
- 主请求函数添加最多3次重试
- 针对403和429错误自动重试
- 智能重试策略，避免过度请求

## 使用方法

### 1. 使用启动脚本（推荐）
```bash
./start.sh
```

### 2. 手动启动
```bash
# 安装依赖
go mod tidy

# 构建项目
go build -o duck2api

# 运行
./duck2api
```

### 3. 测试功能
```bash
# 安装Python和requests库
pip install requests

# 运行测试脚本
python3 test_duck2api.py
```

## 环境变量配置

```bash
# 服务器配置
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"

# 可选：认证
export Authorization="your_auth_key"

# 可选：代理
export PROXY_URL="http://proxy:port"

# 可选：TLS证书
export TLS_CERT="path/to/cert.pem"
export TLS_KEY="path/to/key.pem"
```

## 测试API

### 获取模型列表
```bash
curl http://localhost:8080/v1/models
```

### 聊天完成（非流式）
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

### 聊天完成（流式）
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Count from 1 to 5"}],
    "stream": true
  }'
```

## 支持的模型

- `gpt-4o-mini` - OpenAI GPT-4o Mini
- `o3-mini` - OpenAI O3 Mini
- `claude-3-haiku-20240307` - Anthropic Claude 3 Haiku
- `meta-llama/Llama-3.3-70B-Instruct-Turbo` - Meta Llama 3.3 70B
- `mistralai/Mixtral-8x7B-Instruct-v0.1` - Mistral Mixtral 8x7B

注意：`gpt-3.5-turbo` 已被DuckDuckGo官方移除，现在映射到 `gpt-4o-mini`。

## 故障排除

### 1. 如果仍然出现403错误
- 检查网络连接
- 尝试使用代理：`export PROXY_URL="http://your-proxy:port"`
- 等待一段时间后重试

### 2. 如果出现429错误
- 降低请求频率
- 等待几分钟后重试
- 考虑使用多个代理IP

### 3. 如果Token获取失败
- 检查网络连接到DuckDuckGo
- 确认防火墙没有阻止连接
- 尝试使用VPN或代理

### 4. 如果编译失败
- 确保Go版本 >= 1.21
- 运行 `go mod tidy` 清理依赖
- 检查网络连接，确保能下载Go模块

## 注意事项

1. **合规使用**：请遵守DuckDuckGo的使用条款，避免过度请求
2. **稳定性**：服务稳定性依赖于DuckDuckGo的可用性
3. **更新**：DuckDuckGo可能随时更改其API，需要持续关注更新
4. **代理**：如果频繁出现403错误，建议配置代理池

## 更新日志

- 升级TLS指纹到Chrome 131
- 添加用户代理随机化
- 改进Token获取重试机制
- 增强错误处理和重试策略
- 添加测试脚本和启动脚本
- 更新请求头以减少检测风险