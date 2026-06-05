# GPBX 技术文档 — Go 语言 PBX/呼叫中心系统架构设计与开发指南

[![Go Version](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)]()

## 📖 目录

- [项目概述](#项目概述)
- [为什么选择 GPBX](#为什么选择-gpbx)
- [系统架构设计](#系统架构设计)
- [核心模块详解](#核心模块详解)
- [功能模块详解](#功能模块详解)
- [存储设计](#存储设计)
- [配置系统](#配置系统)
- [呼叫流程详解](#呼叫流程详解)
- [开发指南](#开发指南)
- [部署建议](#部署建议)
- [技术选型对照表](#技术选型对照表)
- [未来演进方向](#未来演进方向)

---

## 项目概述

**GPBX**（Go PBX）是一款基于 **Go 语言** 从零构建的 **企业级 PBX 电话交换系统**，深度集成 **FreeSWITCH 软交换平台**。项目以现代技术栈和清晰的分层架构，提供完整的呼叫中心能力——分机管理、号码路由、IVR 语音导航、AI 智能机器人、Telegram Bot 电话集成等开箱即用。

> 🎯 **一句话总结**：用 Go 写的 FreeSWITCH 管理面板 + 呼叫中心 + AI 电话机器人，三合一。

## 为什么选择 GPBX

### 🆚 与传统 PBX 方案对比

| 对比维度 | 传统 PBX（如 Asterisk GUI） | GPBX |
|---------|---------------------------|------|
| **开发语言** | C/PHP/Python 混合 | 纯 Go，编译为单一二进制 |
| **部署方式** | 依赖多组件，配置复杂 | 单文件部署，零依赖 |
| **API 风格** | 无标准或 SOAP | RESTful API + WebSocket |
| **AI 集成** | 需额外开发 | 内置 OpenAI/Dify 支持 |
| **多租户** | 通常不支持 | 原生多租户隔离 |
| **Telegram Bot** | 不支持 | 内置集成，消息即电话 |
| **性能** | 解释型/多进程 | Go 高并发，goroutine |
| **存储** | 依赖 MySQL/PostgreSQL | SQLite（嵌入式）或文件存储 |

### ✨ 核心优势

1. **Go 语言高并发**：goroutine + channel 原生并发模型，轻松承载数千路通话
2. **事件驱动架构**：EventBus 解耦模块，Pub/Sub + Request/Response 双模式
3. **双存储引擎**：开发用小文件、生产用 SQLite，无需外部数据库
4. **AI 原生集成**：支持 OpenAI 协议（Qwen/DeepSeek 等）和 Dify 工作流编排
5. **Telegram Bot 打电话**：消息界面发送号码即可发起通话，无需专用客户端
6. **模块化可扩展**：清晰的 `Module` 接口，新增功能只需注册模块

---

## 系统架构设计

### 整体分层架构

```
┌─────────────────────────────────────────────┐
│              前端层                        │
│  callcenter_manager_web · callcenter_agent_web  │
└─────────────────────┬───────────────────────┘
                      │ HTTP / WebSocket
┌─────────────────────▼───────────────────────┐
│            HTTP Server（Gin）                 │
│  Auth · Middleware · CORS · JWT            │
└─────────────────────┬───────────────────────┘
                      │
┌─────────────────────▼───────────────────────┐
│             功能模块层（modules/server）        │
│  callcenter_manager · callcenter_agent        │
│  callcenter_api · telegram_bot · ai        │
│  ivr · router · monitor                    │
└─────────────────────┬───────────────────────┘
                      │ EventBus / Direct Call
┌─────────────────────▼───────────────────────┐
│             核心层                              │
│  app（模块生命周期）· datacenter（会话/任务） │
│  event（事件总线）· pbx（呼叫操作）         │
│  esl（FreeSWITCH ESL 客户端）               │
└─────────────────────┬───────────────────────┘
                      │
┌─────────────────────▼───────────────────────┐
│             存储层 · 配置层                    │
│  store（GORM/File）· kcfg（配置解析）       │
│  log（日志）                                 │
└─────────────────────────────────────────────┘
                      │ ESL TCP
┌─────────────────────▼───────────────────────┐
│          FreeSWITCH（电话交换核心）            │
└─────────────────────────────────────────────┘
```

### 模块化设计原则

每个模块实现 `Module` 接口：

```go
type Module interface {
    Init() error
    Run() error
    Uninit() error
}
```

模块通过 `app.GetDefaultApp().Add(priority, module)` 注册，按优先级顺序初始化和启动。模块间通过 `event.EventBus` 松耦合通信，也可直接调用其他模块的导出函数。

---

## 核心模块详解

### 1. app — 应用框架

负责：
- 解析 `config.kcfg` 配置文件
- 按优先级注册和启动所有 `Module`
- 管理模块生命周期（Init → Run → Uninit）
- 提供 `GetStoreEngine()` 全局存储引擎入口

`Run()` 方法阻塞在 `exit_sig` channel 上，直到 `Uninit()` 被调用。

### 2. event — 事件总线

支持两种通信模式：

| 模式 | 方法 | 说明 |
|------|------|------|
| 发布/订阅 | `Publish(topic, data)` / `Subscribe(topic)` | 一对多， fire-and-forget |
| 请求/响应 | `Request(topic, data, timeout)` | 一对一，等待响应 |

关键 Topic：

| Topic | 说明 |
|-------|------|
| `TOPIC_SESSION_CREATE` | 会话创建 |
| `TOPIC_SESSION_ANSWER` | 通话应答 |
| `TOPIC_SESSION_HANGUP` | 通话挂断 |
| `TOPIC_SESSION_DESTROY` | 会话销毁 |
| `TOPIC_SEND_API` | 发送 FreeSWITCH API 命令 |
| `TOPIC_SEND_API_WITH_PROMISE` | 发送 API 并等待结果 |
| `TOPIC_ASR_RESULT` | ASR 识别结果 |

### 3. datacenter — 会话与任务管理

**SessionManager**：管理业务会话，每个会话关联一个 `Task`，包含 A-leg 和 B-leg 信息。

**Task** 类型：

| 类型 | 说明 |
|------|------|
| `originate` | 主动外呼 |
| `callin` | 被动呼入 |
| `to_robot` | 转接到机器人 |
| `to_vendor` | 转接到三方供应商 |
| `to_ivr` | 转接到 IVR |
| `to_user` | 转接到用户分机 |
| `to_acd` | 转接到排队机 |

Task 使用 `TryLock` 模式保证状态转移的线程安全。

### 4. esl — FreeSWITCH ESL 客户端

维护到 FreeSWITCH Event Socket 的 TCP 长连接，支持：
- 异步命令发送（`SendAPI` / `SendAPP`）
- 事件订阅与回调（`On("EVENT_NAME", callback)`）
- 自动重连（`Reconnect_Interval_Second`）

### 5. pbx — 呼叫操作封装

所有呼叫操作通过构造 `PBXCommand` 并发布到 EventBus 实现：

| 函数 | 说明 |
|------|------|
| `OriginateToRobot()` | 外呼转机器人 |
| `OriginateToIvr()` | 外呼转 IVR |
| `Transfer()` | 通话转接 |
| `Playback()` | 播放音频 |
| `HangupCall()` | 挂断通话 |

底层通过 Lua 脚本（`scripts/` 目录）与 FreeSWITCH 交互，支持 `bgapi` 异步执行和 `promise` 机制获取结果。

---

## 功能模块详解（modules/server）

### callcenter_manager — 管理后台 API

提供完整的 RESTful API，用于管理：
- 租户（Tenant）
- 分机（Extension）
- 坐席（Agent）
- 机器人（Robot）
- 租户号码（TenantNumber）
- IVR 配置（Ivr）
- Telegram 用户（TeleGramUser）

所有接口均需 JWT 认证，通过 `auth.Middleware()` 中间件校验。

### callcenter_agent — 坐席客户端

提供 WebSocket 接口（`/agent`），实现坐席的实时操作：
- 分机注册与心跳
- 来电弹屏
- 接听/挂断/转接/播放

### callcenter_api — 呼叫业务 API

对外提供呼叫相关业务接口，供第三方系统调用：
- `POST /api/originate` — 发起外呼
- `POST /api/transfer` — 转接
- `POST /api/playback` — 播放

### telegram_bot — Telegram Bot 集成

通过 Telegram Bot 实现远程呼叫控制：

1. 用户向 Bot 发送电话号码
2. 系统校验用户权限（TeleGramUser 表）
3. 随机选取用户绑定的在线分机
4. 发起 `OriginateToIvr` 呼叫
5. 通话过程中 Bot 可接收 DTMF / Answer / Hangup 回调

每个 Telegram Chat 对应一个 `BotBody`，通过 `globalBotManager` 以 TaskID 为 key 管理生命周期。

关键配置（`config.kcfg`）：

```
telegram_bot {
    token           "Bot Token"
    enable          true
    bind_script     "telegram_bots/10000.lua"
    to_ivrid         10001       # 默认 IVR ID
    tenant_id       20000       # 默认租户 ID
    number          10000       # 默认外显号码
    timeout         10          # 超时（秒）
    proxy           "http://..." # 可选代理
}
```

### ai / ivr — 智能交互

**AI 引擎**（modules/server/ai）：
- 支持 OpenAI 协议（Qwen 等模型）
- 支持 Dify 工作流
- 通过 Lua 脚本引擎（lua_engine.go）实现对话管理

**IVR 引擎**（modules/server/ivr）：
- Lua IVR：执行自定义 Lua 脚本
- Dify IVR：调用 Dify 工作流

---

## 存储设计

### 存储接口

`store.Store` 接口定义了完整的数据操作契约，支持多种后端实现：

| 实现 | 说明 |
|------|------|
| `gormStore` | 基于 GORM + SQLite，推荐生产使用 |
| `kcfgStore` | 基于文件存储，适合嵌入式场景 |
| `memoryStore` | 纯内存存储，用于 kcfgStore 的缓存层 |

### 数据模型关系

```
Tenant 1───* TenantNumber
Tenant 1───* Extension
Tenant 1───* Agent
Tenant 1───* Robot
Tenant 1───* Ivr
Tenant 1───* TeleGramUser

Agent *──1 Extension（通过 ExtensionId 关联）
```

### Extension 在线管理

`ExtensionManagerInstance` 运行时管理分机注册信息（`ExtensionRegisterInfo`），包括：
- `Number` — 分机号码
- `Contact` — 联系地址
- `NetworkIP` / `NetworkPort` — 分机网络地址
- `IsValid()` — 判断是否有效注册

---

## 配置系统（kcfg）

自定义配置格式，支持嵌套结构和多类型解析：

```kcfg
# 注释以 # 开头
freeswitch {
    addr    127.0.0.1:8021
    user    default
    pass    ClueCon
}
```

访问方式：`cfg.Child("freeswitch.addr").GetString()`

---

## 呼叫流程详解

### 外呼转机器人（Originate → Robot）

```
第三方系统
    │ POST /api/originate
    ▼
callcenter_api
    │ 创建 Task
    ▼
pbx.OriginateToRobot()
    │ 发送 Lua 脚本（bgapi originate + park）
    ▼
FreeSWITCH
    │ 发起 SIP 呼叫
    │ 被叫应答
    ▼
ESL Event: CHANNEL_ANSWER
    │
    ▼
esl_notify → Publish TOPIC_SESSION_ANSWER
    │
    ▼
router 模块
    │ 查询 TenantNumber 获取路由
    │ 转接到 Robot
    ▼
AI Engine（Lua/OpenAI/Dify）
    │ 智能对话
    │ ASR 结果通过 robot::asr 事件回传
    ▼
通话结束 → CHANNEL_HANGUP → TOPIC_SESSION_HANGUP
```

### Telegram Bot 发起呼叫

```
Telegram 用户
    │ 发送电话号码
    ▼
BotContext.defaultHandler()
    │ 查询 TeleGramUser 权限
    │ 创建/复用 BotBody
    ▼
BotBody.OnCallNumber()
    │ 随机选取在线分机
    │ 使用用户绑定的 IvrId（或默认 to_ivrid）
    ▼
pbx.OriginateToIvr()
    │
    ▼
FreeSWITCH → 通话建立
    │
    ▼
BotBody.OnAnswer() / OnHangup() / OnDtmf()
    通过 LuaEngine 回调 Telegram 用户
```

---

## 开发指南

### 添加新 HTTP 接口

1. 在对应模块的 `.go` 文件中定义 handler 函数
2. 在 `router.go` 的 `routers` map 或 `registers` slice 中注册路由
3. 使用 `c.ShouldBindJSON()` 解析请求体
4. 使用 `c.JSON()` 返回响应

### 添加新 FreeSWITCH 事件处理

1. 在 `modules/notify/esl_notify/esl.go` 的 `Init()` 中添加 `esl.On("EVENT_NAME", callback)`
2. 在回调中将 FreeSWITCH 事件转换为内部事件结构体
3. 通过 `event.GetDefaultBus().Publish()` 发布到 EventBus
4. 在需要的模块中 `Subscribe` 订阅该事件

### 添加新存储实体

1. 在 `store/store.go` 中定义结构体（添加 gorm tag）
2. 在 `Store` 接口中添加 CRUD 方法声明
3. 在 `gorm_store.go`、`memory_store.go`、`kcfg_store.go` 中实现
4. 在管理后台模块中添加对应 API handler

---

## 部署建议

### 最小化部署（单机）

```bash
# 1. 编译
go build -o gpbx .

# 2. 准备配置
cp config.kcfg.example config.kcfg
# 编辑 config.kcfg 配置 FreeSWITCH 地址和管理员密码

# 3. 启动
./gpbx
```

仅需一个二进制文件 + 一个配置文件即可运行，无需安装数据库、Web 服务器等依赖。

### 生产环境推荐

| 组件 | 推荐方案 | 说明 |
|------|---------|------|
| 进程管理 | systemd 或 supervisor | 守护进程，自动重启 |
| 反向代理 | Nginx + Let's Encrypt | HTTPS 加密，静态文件加速 |
| 数据库备份 | cron + rsync | 定期备份 `.db/gpbx.db` |
| 日志收集 | journald 或 filebeat | 集中采集和监控 |
| 监控告警 | Prometheus + Grafana | （规划中） |

### 安全加固清单

- [ ] 修改默认管理员密码 `admin/admin123`
- [ ] 更换 JWT Secret 为随机字符串（`openssl rand -base64 32`）
- [ ] 启用 HTTPS（配置 `enable_ssl: true`）
- [ ] 使用 `sign_tool` 生成机器授权证书
- [ ] 配置防火墙，仅暴露必要的 HTTP 端口
- [ ] 定期更新 Go 版本和安全补丁

### 运行时目录结构

```
/opt/gpbx/
├── gpbx                       # 编译产物（单一二进制）
├── config.kcfg                # 配置文件
├── .db/
│   └── gpbx.db               # SQLite 数据库（定期备份！）
├── scripts/                   # FreeSWITCH Lua 脚本
│   ├── do_originate_and_park.lua
│   └── do_transfer.lua
├── robots/                    # 机器人 Lua 脚本
├── ivrs/                      # IVR 自定义脚本
├── telegram_bots/             # Telegram Bot 脚本
├── callcenter_manager_web/    # 管理后台前端（静态文件）
└── callcenter_agent_web/      # 坐席前端（静态文件）
```

---

## 技术选型对照表

| 需求 | 选型 | 备选方案 | 选型理由 |
|------|------|---------|---------|
| 开发语言 | **Go 1.26** | Java/Node.js/Python | 高并发、单二进制、低内存 |
| Web 框架 | **Gin 1.12** | Echo/Fiber | 生态成熟，性能优秀 |
| 软交换 | **FreeSWITCH** | Asterisk | ESL 协议更稳定，社区活跃 |
| 数据库 | **SQLite (GORM)** | PostgreSQL/MySQL | 零运维，适合嵌入场景 |
| 配置格式 | **kcfg（自研）** | YAML/TOML | 简洁，支持嵌套和类型解析 |
| 脚本引擎 | **gopher-lua** | Starlark/goja | Lua 5.1 兼容，FreeSWITCH 原生支持 |
| AI 协议 | **OpenAI 兼容** | 自研协议 | 通用性强，可对接多种 LLM |
| AI 工作流 | **Dify** | LangChain/n8n | 可视化编排，中文友好 |
| 即时通讯 | **Telegram Bot** | 微信/钉钉/飞书 | API 开放，无需企业认证 |
| 实时通信 | **WebSocket** | SSE/gRPC Stream | 浏览器原生支持，全双工 |
| 认证 | **JWT** | Session/OAuth2 | 无状态，适合 API 和 WebSocket |

---

## 未来演进方向

- 🐳 **容器化部署**：Docker 镜像 + docker-compose 一键部署方案
- ☸️ **Kubernetes 支持**：Helm Chart，水平扩展
- 🧩 **微服务拆分**：呼叫服务、AI 服务、Bot 服务独立部署
- 📹 **视频通话**：基于 WebRTC 的视频呼叫能力
- 💬 **多渠道接入**：微信企业号、钉钉、飞书、Slack 等
- 📊 **统计分析**：通话报表、坐席绩效、ASR 准确率分析
- 🔌 **插件系统**：Go plugin 或 WASM 沙箱，支持第三方扩展
- 🎛️ **可视化 IVR 编辑器**：拖拽式 IVR 流程设计器
- 📞 **SIP Trunk 管理**：运营商线路对接、费率管理
- 🔐 **国密支持**：SM2/SM3/SM4 加密套件

> 💡 欢迎通过 Issue 提出需求，或直接提交 PR 贡献代码！详见 [CONTRIBUTING](./CONTRIBUTING.md)（规划中）。

---

## 许可证

本项目采用 [MIT License](./LICENSE) 开源协议。你可以自由使用、修改和分发，商业用途友好。

---

<p align="center">
  <sub>
    📖 <a href="https://gitee.com/kolonse_zhjsh/gpbx2/blob/master/readme.md">项目简介</a> ·
    🏗️ <a href="https://gitee.com/kolonse_zhjsh/gpbx2/blob/master/CODEBUDDY.md">开发者指南</a> ·
    💬 <a href="https://gitee.com/kolonse_zhjsh/gpbx2/issues">反馈建议</a> ·
    ⭐ <a href="https://gitee.com/kolonse_zhjsh/gpbx2">给个 Star</a>
  </sub>
</p>

<p align="center">
  <sub>Made with ❤️ by <a href="https://gitee.com/kolonse_zhjsh">kolonse</a></sub>
</p>
