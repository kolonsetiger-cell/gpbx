# GPBX - Go 语言 PBX 电话交换系统 | FreeSWITCH 呼叫中心

[![Go Version](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)]()

## 项目简介

**GPBX** 是一个用 Go 语言开发的企业级 **PBX（Private Branch Exchange，电话交换机）** 系统，深度集成 **FreeSWITCH** 软交换平台，提供开箱即用的 **呼叫中心（Call Center）** 能力。无论你是需要搭建智能客服热线、自动外呼系统，还是希望为 Telegram Bot 添加打电话功能，GPBX 都能满足你的需求。

系统采用**模块化、事件驱动**的设计，支持**多租户隔离**、**AI 智能机器人**（兼容 OpenAI 协议和 Dify 工作流）、**Telegram Bot 呼叫集成**、**IVR 语音导航**等特性，适用于以下场景：

- 🏢 **企业总机/客服热线**：来电自动分配、IVR 语音导航、坐席接听
- 📞 **自动外呼系统**：批量外呼、AI 对话、通话记录追踪
- 🤖 **AI 语音机器人**：对接 LLM 实现智能语音对话
- 💬 **Telegram Bot 电话**：通过 Telegram 消息发起/接听电话
- 🔧 **FreeSWITCH 管理面板**：分机管理、号码路由、在线监控

> 📌 **关键词**：Go PBX, FreeSWITCH 呼叫中心, Go 电话系统, AI 语音机器人, Telegram Bot 电话, IVR 系统, SIP 分机管理, 开源 PBX

## 核心功能

- **通话管理**：呼出、呼入、转接、桥接、播放、DTMF 采集
- **会话管理**：实时跟踪通话会话全生命周期（创建、振铃、应答、挂断、销毁）
- **任务管理**：支持 originate（外呼）、callin（呼入）等多种任务类型
- **多租户架构**：租户隔离，支持租户号码、分机、坐席独立管理
- **分机管理**：SIP 分机注册、在线状态管理、网络地址（NetworkIP/NetworkPort）配置
- **坐席管理**：呼叫中心坐席管理，支持 WebSocket 实时通信
- **号码路由**：支持 gateway / IMS / local 三种路由类型，动作支持转机器人或转三方
- **AI 集成**：支持 OpenAI 协议（Qwen 等）、Dify 工作流
- **Telegram Bot**：通过 Telegram Bot 发起呼叫、接收通话回调（DTMF/Answer/Hangup）
- **IVR 系统**：支持 Lua 脚本 IVR 和 Dify IVR 两种类型
- **HTTP API**：基于 Gin 的 RESTful API，支持 JWT 认证
- **事件总线**：模块间松耦合通信，支持 Pub/Sub 和 Request/Response 模式
- **双存储引擎**：支持文件存储（kcfg）和 SQLite（GORM）两种存储后端

## 系统架构

```
GPBX
├── 核心层
│   ├── app/           # 应用框架，模块生命周期管理
│   ├── datacenter/   # 会话（Session）和任务（Task）管理
│   ├── event/        # 事件总线（EventBus）
│   ├── pbx/          # PBX 核心操作（Originate/Transfer/Playback/Hangup）
│   └── esl/          # FreeSWITCH ESL 客户端
├── 存储层
│   └── store/        # 存储接口，支持 GORM（SQLite）和 kcfg（文件）
├── 配置层
│   └── kcfg/        # 自定义配置格式解析
├── 功能模块 (modules/server/)
│   ├── callcenter_manager/  # 管理后台 API（租户/分机/坐席/机器人/号码/IVR）
│   ├── callcenter_agent/     # 坐席客户端 API + WebSocket
│   ├── callcenter_api/      # 呼叫业务 API（外呼/转接/播放）
│   ├── telegram_bot/       # Telegram Bot 集成
│   ├── ai/                 # AI 引擎（OpenAI/Dify/Lua）
│   ├── ivr/                # IVR 引擎（Lua/Dify）
│   ├── router/              # 呼叫路由（呼入/呼出）
│   └── monitor/             # 系统监控
├── 通知模块 (modules/notify/)
│   └── esl_notify/         # FreeSWITCH 事件 → EventBus 桥接
├── 前端
│   ├── callcenter_manager_web/  # 管理后台前端（Vue）
│   └── callcenter_agent_web/    # 坐席客户端前端（Vue）
└── 脚本
    ├── scripts/         # FreeSWITCH Lua 脚本（originate/park/transfer）
    ├── robots/          # 机器人 Lua 脚本
    └── ivrs/           # IVR Lua 脚本
```

## 技术栈

- **语言**：Go 1.26.2
- **Web 框架**：Gin 1.12.0
- **WebSocket**：gorilla/websocket
- **电话交换**：FreeSWITCH（ESL 集成）
- **脚本引擎**：yuin/gopher-lua（Lua 5.1）
- **存储**：SQLite（GORM v1.31.1）+ 文件存储（kcfg）
- **认证**：JWT（golang-jwt/jwt/v5）
- **Telegram**：go-telegram/bot v1.21.0
- **AI**：OpenAI 协议 / Dify 工作流

## 快速开始

### 环境要求

- Go 1.26.2 或更高版本
- FreeSWITCH（可选，用于实际通话）
- SQLite（使用 GORM 存储时）

### 配置文件

编辑 `config.kcfg`，主要配置项：

```
# FreeSWITCH 连接地址（与 GPBX 一对一关系）
freeswitch {
    addr                        127.0.0.1:8021
    user                       default
    pass                        ClueCon
    Reconnect_Interval_Second   5
}

# HTTP 服务配置
http {
    host                0.0.0.0
    manager_port        8080   # 管理后台端口
    agent_port          8081   # 坐席 WebSocket 端口
    api_port            8082   # 业务 API 端口
    enable_ssl          false
    admin {
        username       admin
        password       admin123
    }
}

# 存储配置（推荐使用 SQLite）
store {
    path    sqlite://.db/gpbx.db
}

# Telegram Bot 配置（可选）
telegram_bot {
    server_port     8083
    token           "你的 Bot Token"
    enable          true
    bind_script     "telegram_bots/10000.lua"
    to_ivrid         10001
    tenant_id       20000
    number          10000
    timeout         10
}

# LLM 配置（AI 机器人）
llm {
    type        openai
    model       qwen2.5-7b-instruct
    baseurl     http://your-llm-host:1234/v1
    token       your-token
    max_history 10
}
```

### 启动系统

```bash
# 安装依赖
go mod tidy

# 启动系统
go run main.go

# 或编译后启动
go build
./gpbx
```

系统将在配置的 HTTP 端口上启动（默认管理后台 8080）。

## API 接口概览

### 认证

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/auth/login` | POST | 用户登录，返回 JWT Token |
| `/api/auth/refresh` | POST | 刷新 Token |

### 租户管理

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/tenant/list` | GET | 获取租户列表（分页） |
| `/api/tenant/create` | POST | 创建租户 |
| `/api/tenant/update` | PUT | 更新租户 |
| `/api/tenant/delete/:tenantId` | DELETE | 删除租户 |

### 分机管理

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/extension/list` | GET | 获取分机列表（分页） |
| `/api/extension/create` | POST | 创建分机 |
| `/api/extension/update` | PUT | 更新分机 |
| `/api/extension/delete/:extensionId` | DELETE | 删除分机 |
| `/fs/directory` | GET/POST | FreeSWITCH 分机注册接口 |

### 坐席管理

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/agent/list` | GET | 获取坐席列表（分页） |
| `/api/agent/create` | POST | 创建坐席 |
| `/api/agent/update` | PUT | 更新坐席 |
| `/api/agent/delete/:agentId` | DELETE | 删除坐席 |
| `/agent` | WebSocket | 坐席实时通信（注册/来电/操作） |

### 机器人管理

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/robot/list` | GET | 获取机器人列表（分页） |
| `/api/robot/create` | POST | 创建机器人 |
| `/api/robot/update/:id` | PUT | 更新机器人 |
| `/api/robot/delete/:id` | DELETE | 删除机器人 |

### IVR 管理

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/ivr/list` | GET | 获取 IVR 列表（分页） |
| `/api/ivr/create` | POST | 创建 IVR |
| `/api/ivr/update` | PUT | 更新 IVR |
| `/api/ivr/delete/:id` | DELETE | 删除 IVR |

### Telegram 用户管理

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/telegram_user/list` | GET | 获取 Telegram 用户列表（分页） |
| `/api/telegram_user/create` | POST | 创建 Telegram 用户 |
| `/api/telegram_user/update` | PUT | 更新 Telegram 用户 |
| `/api/telegram_user/delete/:username` | DELETE | 删除 Telegram 用户 |

### 通话操作

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/originate` | POST | 发起外呼 |
| `/api/transfer` | POST | 转接通话 |
| `/api/playback` | POST | 播放音频 |
| `/api/task/info` | GET | 获取任务信息 |

## 数据模型

| 模型 | 说明 | 关键字段 |
|------|------|----------|
| **User** | 系统用户 | Username, Password, Roles |
| **Tenant** | 租户 | TenantId, TenantName, ExpireTime, DefaultNumber |
| **TenantNumber** | 租户号码 | Number, TenantId, Action, WayType, RobotID |
| **Extension** | SIP 分机 | TenantId, ExtensionId, Password, Status, NetworkIP, NetworkPort |
| **ExtensionRegisterInfo** | 分机注册信息（运行时） | Number, Contact, NetworkIP, NetworkPort, IsValid |
| **Agent** | 坐席 | AgentId, ExtensionId, DisplayNumber |
| **Robot** | 机器人 | RobotID, Target, Arg, Welcome, Prompt, ToVendor |
| **Ivr** | IVR 配置 | IvrID, Type(Lua/Dify), Path, Args |
| **TeleGramUser** | Telegram 用户 | Username, TenantId, BindScript, BindNumbers, IvrId |

## 呼叫流程

### 外呼流程（Originate → Robot）

1. 调用 `POST /api/originate` 发起外呼
2. `pbx.OriginateToRobot()` 通过 ESL 发送 Lua 脚本到 FreeSWITCH
3. FreeSWITCH 发起呼叫（originate）
4. 被叫应答 → FreeSWITCH 发出 `CHANNEL_ANSWER` 事件
5. `esl_notify` 捕获事件，发布 `TOPIC_SESSION_ANSWER` 到 EventBus
6. `router` 模块收到事件，根据号码路由到对应 Robot
7. FreeSWITCH 桥接呼叫到 Robot 会话
8. ASR 结果通过 `robot::asr` 自定义事件回传

### Telegram Bot 呼叫流程

1. 用户通过 Telegram 向 Bot 发送电话号码
2. `BotContext.defaultHandler` 创建/复用 `BotBody`
3. `BotBody.OnCallNumber` 查询用户权限，随机选取在线分机
4. 发起 `OriginateToIvr` 呼叫，将客户来电桥接到 IVR
5. 通话过程中 Telegram Bot 可接收 DTMF/Answer/Hangup 回调

## 部署指南

### 本地开发环境

```bash
# 克隆仓库
git clone https://gitee.com/kolonse_zhjsh/gpbx2.git
cd gpbx2

# 安装 Go 依赖
go mod tidy

# 修改 config.kcfg 中的 FreeSWITCH 连接地址
# 编辑 freeswitch.addr 指向你的 FreeSWITCH ESL 端口

# 启动服务
go run main.go
```

启动后访问 `http://localhost:8080` 进入管理后台，默认账号 `admin` / `admin123`。

### 生产环境部署

```bash
# 编译二进制
go build -o gpbx .

# 直接运行
./gpbx

# 推荐使用 systemd 管理进程（创建 /etc/systemd/system/gpbx.service）
```

**systemd 服务示例**：

```ini
[Unit]
Description=GPBX Call Center Service
After=network.target freeswitch.service

[Service]
Type=simple
User=gpbx
WorkingDirectory=/opt/gpbx
ExecStart=/opt/gpbx/gpbx
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker 部署（规划中）

> Docker 镜像和 docker-compose 方案正在开发中，欢迎贡献。

## 安全注意事项

- 默认管理员账号：`admin` / `admin123`，**生产环境请务必修改**
- 配置文件中的 JWT Secret 请更换为随机字符串
- 建议使用 HTTPS（配置 `enable_ssl: true` + 证书）
- 使用 `sign_tool` 进行授权管理：
  - `gpbx -info` 生成机器指纹
  - `sign_tool -sign` 生成授权证书
  - `gpbx -reg` 注册授权证书

## 常见问题 (FAQ)

<details>
<summary><strong>GPBX 和 FreeSWITCH 是什么关系？</strong></summary>
<br>
GPBX 不替代 FreeSWITCH，而是作为 FreeSWITCH 的上层管理系统。GPBX 通过 ESL（Event Socket Library）协议与 FreeSWITCH 通信，管理分机、路由、IVR 等业务逻辑，FreeSWITCH 负责底层的 SIP 信令和媒体处理。两者是一对一关系。
</details>

<details>
<summary><strong>支持哪些 AI 模型？</strong></summary>
<br>
目前支持两种方式：1) <strong>OpenAI 协议</strong>（可对接 Qwen、DeepSeek 等兼容 API）；2) <strong>Dify 工作流</strong>（可视化 AI 应用编排）。机器人通过 ASR（语音识别）将用户语音转文字，发给 AI 模型处理，再将回复通过 TTS 播放给用户。
</details>

<details>
<summary><strong>Telegram Bot 如何打电话？</strong></summary>
<br>
用户向 Bot 发送电话号码，Bot 通过 FreeSWITCH 发起外呼，将用户和 Bot 管理员桥接。通话过程中 Bot 可接收 DTMF、Answer、Hangup 等事件回调。详见 <a href="#telegram-bot-呼叫流程">Telegram Bot 呼叫流程</a>。
</details>

<details>
<summary><strong>如何贡献代码？</strong></summary>
<br>
欢迎提交 Issue 和 Pull Request。开发前请先阅读 <code>CODEBUDDY.md</code> 了解项目架构和编码规范。
</details>

## 许可证

本项目采用 [MIT License](./LICENSE) 开源协议。你可以自由使用、修改和分发，但需保留原始版权声明。

---

<p align="center">
  <sub>Made with ❤️ by <a href="https://gitee.com/kolonse_zhjsh">kolonse</a> | 
  ⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！</sub>
</p>
