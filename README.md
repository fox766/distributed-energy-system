# 分布式能源交易系统

基于 Hyperledger Fabric 的分布式能源交易平台，支持 P2P 绿电交易、碳减排追踪、分时电价和自动撮合。

## 核心特性

| 特性 | 说明 |
|------|------|
| **P2P 能源交易** | 生产者售电、消费者购电，链上撮合结算 |
| **碳减排追踪** | 按能源类型（太阳能 0.6 / 风能 0.7 / 储能 0.5 kg/kWh）计算碳减排量，不可篡改 |
| **分时电价 (TOU)** | 高峰 ¥1.2（9-12h, 17-21h）、平段 ¥0.8、低谷 ¥0.4，订单可自动采用当前电价 |
| **自动撮合引擎** | 创建订单自动撮合，支持交付时段匹配和部分成交（残量订单拆分） |
| **自动发电调度** | 虚拟智能电表模拟光伏/风电/储能的发电曲线，每 30 秒自动发电 |
| **不可篡改审计** | 每笔操作记录在 Fabric 账本，包含交易哈希 |
| **交易账单** | 月度收支汇总、历史账单对比、碳减排统计 |

## 技术架构

```
浏览器 (Vue 3 SPA)
    │
    ├─ http://localhost/index.html
    │
▼ nginx / Apache
    │
    ├─ /api/* ──────────────────────────────────────┐
    │                                               │
▼ Go 后端 (Gin + JWT)                              │
    │                                               │
    ├─ handler/   REST API 层                       │
    ├─ middleware/ JWT 认证 + 角色鉴权              │
    ├─ store/     SQLite 用户凭证存储               │
    └─ fabric/    Hyperledger Fabric Gateway SDK    │
                      │                             │
                      ▼                             │
  ┌──────────────────────────────────────┐         │
  │     Hyperledger Fabric Network       │         │
  │  ┌─────────┐  ┌─────────┐           │         │
  │  │ Peer0   │  │ Peer0   │           │         │
  │  │ Org1    │  │ Org2    │           │         │
  │  └────┬────┘  └────┬────┘           │         │
  │       └──────┬─────┘                 │         │
  │         Orderer (Raft)               │         │
  │              │                       │         │
  │     Chaincode "basic" (Go)          │         │
  │     - 用户/订单/撮合/碳追踪          │         │
  └──────────────────────────────────────┘         │
```

### 技术栈

| 层 | 技术 |
|------|------|
| 区块链 | Hyperledger Fabric 2.5, fabric-contract-api-go v2 |
| 后端 | Go 1.21+, Gin, JWT (HS256), fabric-gateway v1.8 |
| 存储 | SQLite (modernc.org/sqlite, 纯 Go 无 CGO) |
| 前端 | Vue 3 CDN, Chart.js 4, Font Awesome 6 |
| 部署 | Docker Compose, nginx |

## 快速启动

### 前置依赖

- Go 1.21+
- Docker & Docker Compose
- jq
- make (可选)

### 1. 启动 Fabric 网络

```bash
cd blockchain/network
bash start.sh
```

首次启动需要拉取 Fabric 镜像并生成加密材料，约 2-3 分钟。

### 2. 启动后端

```bash
cd application/backend
go run main.go
```

后端启动后会自动：
- 创建/打开 SQLite 数据库 `credentials.db`
- 初始化管理员账户（admin / admin123）
- 连接 Fabric 网络并初始化链码状态
- 启动发电调度器（30s 间隔）和自动撮合调度器（15s 间隔）

### 3. 访问

打开浏览器访问 **http://localhost/index.html**

### 4. 停止

```bash
cd blockchain/network
bash stop.sh
# 停止后端: fuser -k 8080/tcp
```

### Docker 部署（可选）

```bash
docker compose up -d
```

后端暴露在 `:8080`，前端通过 nginx 暴露在 `:80`。

## API 文档

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/register` | 注册用户（username, password, role=PRODUCER\|CONSUMER, energyType），赠送初始余额（`INITIAL_BALANCE`，默认 ¥1000） |
| POST | `/api/login` | 登录，返回 JWT token |
| GET | `/api/market/status` | 系统状态（电价、用户数、订单数） |
| GET | `/api/market/tou-price` | 当前分时电价和时段 |
| GET | `/api/market/price-history` | 历史电价走势 |
| GET | `/api/market/carbon-stats` | 碳减排统计 |
| GET | `/api/market/audit-log` | 审计日志 |
| GET | `/api/orders?status=ALL` | 全部订单 |
| GET | `/api/orders/:id` | 订单详情 |
| GET | `/api/transactions?userid=xxx&month=2026-08` | 交易记录 |
| GET | `/api/transactions/summary?userid=xxx&month=2026-08` | 月度账单 |
| GET | `/api/transactions/statement?userid=xxx` | 全部历史账单 |

### 需认证（Bearer Token）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/user/me` | 当前用户信息 |
| PUT | `/api/user/me` | 更新用户信息（available/balance 均仅 admin 可设；非 admin 一律 403） |
| POST | `/api/orders` | 创建订单（price=0 自动使用 TOU 电价） |
| GET | `/api/orders/mine` | 我的订单 |
| POST | `/api/orders/:id/match` | 手动配对两笔订单（body: `{"counterpartyOrderId":"..."}`） |
| POST | `/api/orders/:id/auto-match` | 自动撮合此订单（撮合即结算，CREATED→FINISHED） |
| POST | `/api/orders/:id/settle` | 结算 MATCHED 订单对（整对原子结算，第二次结算被拒） |
| POST | `/api/orders/:id/cancel` | 取消订单（需为创建者） |
| POST | `/api/generate` | 手动发电（仅 PRODUCER/admin；deviceType 须为 SOLAR_PANEL/WIND_TURBINE/BATTERY_STORAGE） |

### 管理员接口（需 admin 角色）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/energy-price` | 更新电价 |
| POST | `/api/admin/auto-match` | 批量撮合所有待撮合订单（循环排空账本，返回撮合对数） |
| POST | `/api/admin/user/top-up` | 给用户充值（body: `{"userid":"...","amount":500}`） |

### 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `PORT` | `8080` | 后端监听端口 |
| `INITIAL_BALANCE` | `1000` | 新用户注册时的初始余额 |
| `DISABLE_SCHEDULERS` | 未设置 | 设置为任意值可关闭自动发电/自动撮合调度器（便于确定性测试） |
| `FABRIC_PEER_ADDRESS` | `dns:///localhost:7051` | Peer 地址（Docker 部署时改） |
| `JWT_SECRET` / `ADMIN_USERNAME` / `ADMIN_PASSWORD` | — | 安全相关配置 |

### 撮合与结算模型

- **主链路（自动撮合）**：创建订单后由 15s 调度器或 `POST /api/orders/:id/auto-match` 撮合。撮合即结算：一次交易内完成转账、交付，订单直接 CREATED→FINISHED，部分成交自动拆分残单。
- **手动链路**：`POST /api/orders/:id/match` 显式配对两笔订单（互相记录 `matchedWith`），随后任一方 `settle` 都会**原子结算整对**——第二次结算会被拒绝，同一笔交易不可能重复结算。
- **调度器实现细节**：链码一次只撮合一对（Fabric 无 read-your-writes，批量结算会读到旧状态），后端循环调用直到账本排空。

## 项目结构

```
├── application/
│   ├── backend/                  # Go 后端
│   │   ├── main.go               # 入口，路由注册，调度器
│   │   ├── config/config.go      # 环境变量配置
│   │   ├── fabric/fabric.go      # Fabric Gateway 连接
│   │   ├── handler/              # 请求处理
│   │   │   ├── auth.go           # 注册/登录/管理员引导
│   │   │   ├── order.go          # 订单 CRUD + 撮合
│   │   │   ├── market.go         # 行情/TOU/发电/交易记录
│   │   │   └── user.go           # 用户信息/更新
│   │   ├── middleware/auth.go    # JWT 认证 + 角色鉴权
│   │   ├── model/types.go        # 请求/响应 DTO
│   │   ├── store/store.go        # SQLite 凭证存储
│   │   └── Dockerfile            # 多阶段构建
│   └── www/index.html            # Vue 3 SPA 前端
├── blockchain/
│   ├── chaincode-go/chaincode/   # Go 链码
│   │   ├── module.go             # 数据结构 & 常量
│   │   ├── smartcontract.go      # 业务逻辑（~1200 行）
│   │   └── smartcontract_test.go # 单元测试
│   └── network/                  # Fabric 测试网络
│       ├── start.sh / stop.sh    # 启停脚本
│       └── network.config        # 网络参数
├── docker-compose.yml            # Docker 编排
├── nginx.conf                    # 前端 nginx 配置
└── README.md
```

## 链码 API

链码提供 25 个交易/查询函数：

| 分类 | 函数 | 类型 |
|------|------|------|
| 初始化 | `Init`, `InitTOUSchedule` | 写 |
| 用户 | `RegisterUser`, `GetUser`, `UpdateUser` | 写/读 |
| 订单 | `CreateOrder`, `MatchOrder`, `SettleOrder`, `CancelOrder` | 写 |
| 查询 | `GetOrder`, `GetAllOrders`, `GetUserOrders`, `GetSystemCounts` | 读 |
| 市场 | `GetEnergyStatus`, `UpdateEnergyPrice`, `GetPriceHistory` | 读/写 |
| 碳追踪 | `GetCarbonStats`, `GetCarbonHistory`, `recordCarbon` | 读/内部 |
| 审计 | `GetAuditLog`, `recordAudit` | 读/内部 |
| **P0** | `GetTimeOfUsePrice`, `GenerateEnergy`, `GetGenerationHistory` | 读/写 |
| **P2** | `AutoMatchOrder`, `RunAutoMatch` | 写 |
| **P3** | `GetTransactionHistory`, `GetMonthlySummary`, `GetUserStatement` | 读 |

## 设计决策

### 安全模型

- JWT 密钥通过环境变量 `JWT_SECRET` 配置，默认值仅供开发
- `AdminRequired` 中间件保护管理员接口
- 订单操作有所有权校验（取消需创建者、结算需交易方）
- 用户**不可**自改余额（只能通过交易变动）
- 发电接口限制 PRODUCER/admin 角色调用（链码层二次校验）

### 数据持久化

- 用户凭证存储在 **SQLite**（WAL 模式），重启不丢失
- 链上数据（用户状态、订单、碳记录、审计日志）存储在 Fabric 世界状态
- `Init` 幂等：已初始化的数据不会被重启覆盖
- 用户 ID 和订单 ID 使用 **UUID** 生成，避免时间戳碰撞

### 撮合引擎

- 价格优先：买方匹配最低卖价，卖方匹配最高买价
- **交付窗口重叠**加分（重叠窗口的候选订单优先匹配）
- **部分成交**：金额不匹配时自动拆分大订单，创建残量订单继续挂单
- 调度器每 15 秒批量撮合，订单创建时即时撮合

### 已知限制

| 限制 | 影响 | 改进方向 |
|------|------|----------|
| 计数器非原子 | 高并发下可能丢计数 | 改用 `GetState` + `PutState` 在单个交易内完成 |
| 全量查询无分页 | 数据量大时查询超时 | 添加基于 CouchDB 的范围查询和分页 |
| 单 peer 连接 | peer 故障时服务不可用 | 配置多 peer 故障转移 |
| 前端单文件 | 维护困难 | 拆分为 Vue SFC 组件 |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 后端端口 |
| `JWT_SECRET` | `change-me-in-production` | JWT 签名密钥 |
| `ADMIN_USERNAME` | `admin` | 管理员用户名 |
| `ADMIN_PASSWORD` | `admin123` | 管理员密码 |
| `FABRIC_CRYPTO_PATH` | `../../blockchain/network/organizations/peerOrganizations/org1.example.com` | Fabric 证书路径 |
| `FABRIC_PEER_ADDRESS` | `dns:///localhost:7051` | Fabric peer 地址 |
| `CHAINCODE_NAME` | `basic` | 链码名 |
| `CHANNEL_NAME` | `mychannel` | 通道名 |

## 许可

用于学习和研究目的。
