# duck2api

> ⚠️ **重要更新**: 如果你遇到Duck2api无法使用的问题，请使用修复版本！
> 
> 📋 **修复版本说明**: 查看 [FIXES.md](FIXES.md) 了解详细修复内容
> 
> 🚀 **快速启动**: 运行 `./start.sh` 一键启动修复版本

# Web端 

访问http://你的服务器ip:8080/web

![web使用](https://fastly.jsdelivr.net/gh/xiaozhou26/tuph@main/images/%E5%B1%8F%E5%B9%95%E6%88%AA%E5%9B%BE%202024-04-07%20111706.png)

## Deploy


### 编译部署

#### 标准部署
```bash
git clone https://github.com/yaney01/Duck2api
cd Duck2api
go build -o duck2api
chmod +x ./duck2api
./duck2api
```

#### 修复版本快速启动（推荐）
```bash
git clone https://github.com/yaney01/Duck2api
cd Duck2api
./start.sh
```

#### 功能测试
```bash
# 安装Python和requests库
pip install requests

# 运行测试脚本
python3 test_duck2api.py
```

### Docker部署
## Docker部署
您需要安装Docker和Docker Compose。

```bash
docker run -d \
  --name duck2api \
  -p 8080:8080 \
  ghcr.io/aurora-develop/duck2api:latest
```

## Docker Compose部署
创建一个新的目录，例如duck2api，并进入该目录：
```bash
mkdir duck2api
cd duck2api
```
在此目录中下载库中的docker-compose.yml文件：

```bash
docker-compose up -d
```

## Usage

```bash
curl --location 'http://你的服务器ip:8080/v1/chat/completions' \
--header 'Content-Type: application/json' \
--data '{
     "model": "gpt-4o-mini",
     "messages": [{"role": "user", "content": "Say this is a test!"}],
     "stream": true
   }'
```

## 支持的模型

- ~~gpt-3.5-turbo~~  duckduckGO官方已移除3.5模型的支持  
- claude-3-haiku
- llama-3.3-70b
- mixtral-8x7b
- gpt-4o-mini
- o3-mini
## 🚨 故障排除

如果遇到以下问题：

### 403 Forbidden 错误
- 使用修复版本：`./start.sh`
- 配置代理：`export PROXY_URL="http://your-proxy:port"`
- 等待一段时间后重试

### 429 Rate Limited 错误
- 降低请求频率
- 等待几分钟后重试
- 考虑使用代理池

### Token获取失败
- 检查网络连接
- 确认能访问DuckDuckGo
- 尝试使用VPN

### 编译失败
- 确保Go版本 >= 1.21
- 运行 `go mod tidy` 清理依赖
- 检查网络连接

详细修复说明请查看 [FIXES.md](FIXES.md)

## 高级设置

默认情况不需要设置，除非你有需求

### 环境变量
```

Authorization=your_authorization  用户认证 key。
TLS_CERT=path_to_your_tls_cert 存储TLS（传输层安全协议）证书的路径。
TLS_KEY=path_to_your_tls_key 存储TLS（传输层安全协议）证书的路径。
PROXY_URL=your_proxy_url 添加代理池来。
```

## 鸣谢

感谢各位大佬的pr支持，感谢。


## 参考项目


https://github.com/xqdoo00o/ChatGPT-to-API

## License

MIT License
