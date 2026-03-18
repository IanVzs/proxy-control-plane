# 代理控制平面 (Proxy Control Plane)

一个用于管理和控制多个代理服务的 Web 应用程序，支持多组代理服务、国家过滤、服务统计、权重分配等功能。

## 功能特性

- ✅ **多代理提供商管理** - 支持添加、编辑、删除多个代理提供商
- ✅ **动态模板变量** - 支持在用户名/密码中使用变量，实现动态代理配置
- ✅ **国家支持配置** - 为每个提供商配置支持的国家列表
- ✅ **服务配置管理** - 创建服务并关联多个代理提供商
- ✅ **权重分配** - 为服务配置各代理提供商的权重，实现负载均衡
- ✅ **代理分配接口** - 根据权重返回代理信息供服务使用
- ✅ **每日使用统计** - 统计每个服务每天使用各代理的次数
- ✅ **数据自动清理** - 自动清理超过一个月的使用数据
- ✅ **透明代理接口** - 通过 HTTP 请求头传递代理参数
- ✅ **Web 管理界面** - 优雅的前端界面进行配置管理
- ✅ **本地文件存储** - 配置自动保存到 JSON 文件
- ✅ **Swagger API 文档** - 内置 Swagger UI，方便 API 测试和文档查看

## 技术栈

- **后端**: Go + Gin
- **前端**: HTML + CSS + JavaScript (原生)
- **存储**: 本地 JSON 文件

## 快速开始

### 1. 编译项目

```bash
go build -o proxy-control-plane.exe ./cmd/server
```

### 2. 生成 Swagger 文档（可选）

如果修改了 API 注释，需要重新生成 Swagger 文档：

```bash
# 安装 swag 工具（首次使用）
go install github.com/swaggo/swag/cmd/swag@v1.8.12

# 生成文档
swag init -g cmd/server/main.go -o ./docs
```

### 3. 运行服务

```bash
./proxy-control-plane.exe
```

默认监听端口 `8080`，可通过环境变量 `ADDR` 修改：

```bash
ADDR=:9000 ./proxy-control-plane.exe
```

### 4. 访问 Web 界面

打开浏览器访问: `http://localhost:8080`

### 5. 访问 Swagger API 文档

打开浏览器访问: `http://localhost:8080/swagger/index.html`

Swagger UI 提供了完整的 API 文档和在线测试功能。

## 配置文件

### providers.json (代理提供商配置)

```json
[
  {
    "id": "provider-1",
    "name": "代理提供商A",
    "type": "http",
    "host": "proxy.example.com",
    "port": 8080,
    "username": "user-{country}",
    "password": "pass-{country}-{session}",
    "countries": ["CN", "US", "HK"],
    "enabled": true
  }
]
```

**字段说明:**
- `id`: 唯一标识符（自动生成）
- `name`: 提供商名称
- `type`: 代理类型（http/https）
- `host`: 代理服务器地址
- `port`: 代理服务器端口
- `username`: 认证用户名（支持模板变量）
- `password`: 认证密码（支持模板变量）
- `countries`: 支持的国家代码列表（留空表示支持所有国家）
- `enabled`: 是否启用

**模板变量说明:**

用户名和密码字段支持以下模板变量，在请求时会被动态替换：

| 变量 | 说明 | 示例 |
|------|------|------|
| `{country}` | 国家代码 | CN, US, HK, JP |
| `{region}` | 区域代码 | beijing, tokyo |
| `{session}` | 会话ID | abc123 |
| `{duration}` | 代理时长（分钟） | 1, 5, 10 |

**示例配置:**

```json
{
  "username": "5680765-1465133e",
  "password": "f1d28731-{country}-09768516-5m"
}
```

当请求时传入 `X-Proxy-Country: JP`，密码会被替换为 `f1d28731-jp-09768516-5m`。

### services.json (服务配置)

```json
[
  {
    "name": "service-1",
    "enabled": true,
    "providers": [
      {
        "provider_id": "provider-1",
        "weight": 3
      },
      {
        "provider_id": "provider-2",
        "weight": 1
      }
    ],
    "request_count": 100,
    "last_request_at": 1710000000
  }
]
```

**字段说明:**
- `name`: 服务名称
- `enabled`: 是否启用
- `providers`: 关联的代理提供商列表
  - `provider_id`: 提供商ID
  - `weight`: 权重（权重越高，被选中概率越大）
- `request_count`: 总请求次数
- `last_request_at`: 最后请求时间（Unix 时间戳）

### daily_usage.json (每日使用统计)

```json
[
  {
    "date": "2024-03-15",
    "service": "service-1",
    "provider_id": "provider-1",
    "count": 50
  }
]
```

系统会自动清理超过30天的数据。

## API 接口文档

### 1. 代理分配接口（按权重）

**请求地址:** `GET /api/assign`

**请求参数:**
- `service` (必需): 服务名称

**示例:**

```bash
curl "http://localhost:8080/api/assign?service=my-service"
```

**响应:**

```json
{
  "service": "my-service",
  "provider_id": "provider-1",
  "provider": "代理提供商A",
  "proxy_url": "",
  "host": "proxy.example.com",
  "port": 8080,
  "username": "user-xxx",
  "password": "pass-xxx"
}
```

**说明:**
- 根据服务配置的权重随机选择一个代理提供商
- 自动记录今日使用次数
- 服务必须已启用且关联了至少一个代理提供商

### 2. 透明代理接口

**请求地址:** `GET/POST/PUT/DELETE /proxy`

**请求头:**
- `X-Target-Url` (必需): 目标 URL
- `X-Service-Name` (可选): 服务名称，用于统计和控制
- `X-Proxy-Country` (可选): 期望的国家代码，用于选择提供商和替换 `{country}` 变量
- `X-Proxy-Region` (可选): 区域代码，用于替换 `{region}` 变量
- `X-Proxy-Session` (可选): 会话ID，用于替换 `{session}` 变量
- `X-Proxy-Duration` (可选): 代理时长，用于替换 `{duration}` 变量
- `X-Proxy-Provider` (可选): 指定的提供商 ID

**示例:**

```bash
curl -H "X-Target-Url: https://api.example.com/data" \
     -H "X-Service-Name: my-service" \
     -H "X-Proxy-Country: HK" \
     http://localhost:8080/proxy
```

**动态变量示例:**

```bash
# 使用国家代码动态替换密码中的 {country}
curl -H "X-Target-Url: https://ipinfo.io" \
     -H "X-Proxy-Country: JP" \
     http://localhost:8080/proxy

# 使用多个动态变量
curl -H "X-Target-Url: https://ipinfo.io" \
     -H "X-Proxy-Country: US" \
     -H "X-Proxy-Region: california" \
     -H "X-Proxy-Session: abc123" \
     -H "X-Proxy-Duration: 5" \
     http://localhost:8080/proxy
```

**响应:**
- 成功: 返回目标服务器的响应（透传）
- 失败: 返回 JSON 错误信息

### 3. 代理提供商管理 API

**获取所有提供商:**
```bash
GET /api/providers
```

**添加提供商:**
```bash
POST /api/providers
Content-Type: application/json

{
  "name": "提供商名称",
  "type": "http",
  "host": "proxy.example.com",
  "port": 8080,
  "username": "user",
  "password": "pass",
  "countries": ["CN", "US"],
  "enabled": true
}
```

**更新提供商:**
```bash
PUT /api/providers/{id}
Content-Type: application/json
```

**删除提供商:**
```bash
DELETE /api/providers/{id}
```

### 4. 服务管理 API

**获取所有服务:**
```bash
GET /api/services
```

响应包含服务列表和今日使用统计：
```json
{
  "services": [...],
  "today_usage": {
    "service-1": 50,
    "service-2": 30
  }
}
```

**创建服务:**
```bash
POST /api/services
Content-Type: application/json

{
  "name": "my-service"
}
```

**获取单个服务详情:**
```bash
GET /api/services/{name}
```

**删除服务:**
```bash
DELETE /api/services/{name}
```

**启用/禁用服务:**
```bash
PUT /api/services/{name}/enabled
Content-Type: application/json

{
  "enabled": true
}
```

**为服务添加代理提供商:**
```bash
POST /api/services/{name}/providers
Content-Type: application/json

{
  "provider_id": "provider-1",
  "weight": 3
}
```

**移除服务的代理提供商:**
```bash
DELETE /api/services/{name}/providers/{providerId}
```

**更新代理提供商权重:**
```bash
PUT /api/services/{name}/providers/{providerId}/weight
Content-Type: application/json

{
  "weight": 5
}
```

**获取服务使用统计:**
```bash
GET /api/services/{name}/usage
```

## 使用方式

### 1. 添加代理提供商

1. 打开 Web 界面
2. 切换到"代理提供商"标签
3. 点击"添加提供商"按钮
4. 填写代理服务器信息（包括账号密码）
5. 配置支持的国家（留空表示全部支持）
6. 点击"保存"

### 2. 创建服务并配置代理

1. 切换到"服务配置"标签
2. 点击"添加服务"按钮
3. 输入服务名称并创建
4. 点击服务行的"配置"按钮
5. 点击"添加代理"选择要关联的代理提供商
6. 设置各代理的权重
7. 启用/禁用服务

### 3. 使用代理分配接口

服务可以通过 `/api/assign` 接口获取代理信息：

```bash
# 获取代理信息
curl "http://localhost:8080/api/assign?service=my-service"

# 响应示例
{
  "host": "proxy.example.com",
  "port": 8080,
  "username": "user-xxx",
  "password": "pass-xxx"
}
```

服务可以使用返回的代理信息发起请求：

```python
import requests

# 获取代理
resp = requests.get('http://localhost:8080/api/assign?service=my-service')
proxy_info = resp.json()

# 使用代理发起请求
proxies = {
    'http': f"http://{proxy_info['username']}:{proxy_info['password']}@{proxy_info['host']}:{proxy_info['port']}",
    'https': f"http://{proxy_info['username']}:{proxy_info['password']}@{proxy_info['host']}:{proxy_info['port']}"
}
response = requests.get('https://httpbin.org/ip', proxies=proxies)
print(response.json())
```

### 4. 使用透明代理接口

**方式一：通过 HTTP 请求头**

```bash
curl -H "X-Target-Url: https://example.com" \
     -H "X-Proxy-Country: HK" \
     http://localhost:8080/proxy
```

**方式二：指定特定提供商**

```bash
curl -H "X-Target-Url: https://example.com" \
     -H "X-Proxy-Provider: provider-1" \
     http://localhost:8080/proxy
```

**方式三：指定服务名称**

```bash
curl -H "X-Target-Url: https://example.com" \
     -H "X-Service-Name: my-service" \
     http://localhost:8080/proxy
```

## 权重分配算法

系统使用加权随机算法选择代理提供商：

- 权重越高，被选中的概率越大
- 例如：代理A权重3，代理B权重1，则代理A被选中概率为75%，代理B为25%
- 每次调用 `/api/assign` 接口都会根据权重重新选择

## 配置说明

### 国家过滤

当请求时指定了 `X-Proxy-Country` 参数，系统会：
1. 查找支持该国家的可用提供商
2. 如果没有匹配的提供商，返回 `no_available_provider` 错误
3. 如果提供商的 `countries` 为空数组，表示支持所有国家

### 服务控制

当请求时指定了 `X-Service-Name` 参数，系统会：
1. 检查该服务是否启用
2. 如果禁用，返回 `service_disabled` 错误
3. 如果启用，自动增加请求计数

### 代理选择策略

**透明代理接口 (/proxy):**
1. 优先使用 `X-Proxy-Provider` 指定的提供商
2. 否则查找支持指定国家的提供商
3. 否则查找任意可用的提供商
4. 如果都没有，返回错误

**代理分配接口 (/api/assign):**
1. 根据服务配置的代理列表
2. 按权重随机选择一个代理
3. 返回代理详细信息

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `ADDR` | `:8080` | 服务监听地址 |
| `PROVIDERS_CONFIG` | `providers.json` | 代理提供商配置文件路径 |
| `SERVICES_CONFIG` | `services.json` | 服务配置文件路径 |
| `USAGE_CONFIG` | `daily_usage.json` | 每日使用统计文件路径 |

## 部署说明

### 本地部署

```bash
# 编译
go build -o proxy-control-plane.exe ./cmd/server

# 运行
./proxy-control-plane.exe
```

### Docker 部署

创建 `Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o proxy-control-plane ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/proxy-control-plane .
COPY --from=builder /app/providers.json .
COPY --from=builder /app/services.json .
EXPOSE 8080
CMD ["./proxy-control-plane"]
```

### 生产环境建议

1. **配置文件备份** - 定期备份配置文件
2. **环境变量** - 使用环境变量管理敏感信息
3. **防火墙** - 限制访问 IP
4. **HTTPS** - 生产环境建议使用 HTTPS
5. **日志** - 添加日志记录功能

## 安全建议

⚠️ **重要提示:**
- 代理提供商的账号密码存储在 `providers.json` 文件中，请妥善保管
- 建议使用环境变量或密钥管理服务存储敏感信息
- 生产环境建议启用 HTTPS
- 限制 Web 管理界面的访问权限

## 故障排查

### 问题：无法连接代理服务器
- 检查代理服务器地址和端口是否正确
- 验证用户名和密码是否正确
- 检查网络连接

### 问题：找不到可用的代理提供商
- 确认至少有一个启用的提供商
- 检查国家代码是否匹配
- 查看提供商的 `countries` 配置

### 问题：服务被禁用
- 检查服务的 `enabled` 字段
- 在 Web 界面启用该服务

### 问题：代理分配失败
- 确认服务已创建
- 确认服务已关联代理提供商
- 确认服务已启用

## 许可证

MIT License
