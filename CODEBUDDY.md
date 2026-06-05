# CODEBUDDY.md This file provides guidance to CodeBuddy when working with code in this repository.

## Build and Run Commands

### Build the project
```
go build
```
Compiles all Go modules into a single binary.

### Run the project
```
go run main.go
```
Uses config.kcfg from the current directory. Requires FreeSWITCH ESL connection configured.

### Install dependencies
```
go mod tidy
```
Resolves and downloads all module dependencies.

### Run tests
```
go test ./...
```
Runs all tests in the project.

### Run tests for a specific module
```
go test ./kcfg/...
```
Tests for kcfg module. Available test packages: kcfg, modules/server/ai

## Architecture Overview

GPBX is a Go-based PBX (telephone exchange) system that integrates with FreeSWITCH via ESL for call control. The architecture follows a modular, event-driven design.

### Core Application Framework (app/)

The `App` struct manages module lifecycle. Modules implement the `Module` interface with `Init()`, `Run()`, and `Uninit()` methods. All modules register themselves via `init()` using `app.GetDefaultApp().Add(priority, module)`. The App loads config from kcfg format and initializes a storage engine for persistence.

### Event-Driven Communication (event/)

The internal `EventBus` enables loose coupling between modules. Key topics:
- `TOPIC_SESSION_CREATE/DESTROY/HANGUP`: Session lifecycle events
- `TOPIC_SESSION_ANSWER/STATE`: Call state changes
- `TOPIC_SEND_API/APP`: FreeSWITCH commands
- `TOPIC_ASR_RESULT`: Speech recognition results

Publishers use `GetDefaultBus().Publish(topic, data)`. Subscribers receive via channels from `Subscribe(topic)`. Request/Response pattern available via `Request(topic, data)`.

### FreeSWITCH Integration (esl/, modules/notify/esl_notify/)

The `esl.Client` maintains a TCP connection to FreeSWITCH's Event Socket Library. It handles authentication, event subscription, and command sending (SendAPP for blocking, SendAPI for background).

`esl_notify` module bridges ESL events to the internal EventBus. It subscribes to CHANNEL_* events and CUSTOM events (promise, robot::asr), converting FreeSWITCH variables like `variable_task_id`, `variable_robot` into typed event structs.

### Session and Task Management (datacenter/)

Two session managers track calls:
- `SessionManager`: Business sessions with Task/A_Session/B_Session relationships
- `GlobalSessionManager`: Raw sessions from FreeSWITCH

`Task` represents a call flow with A-leg and B-leg sessions. Task types distinguish originate vs callin flows and their routing destinations (robot, vendor, AI, IVR, user, ACD). Tasks use a TryLock pattern for thread-safe state transitions.

### PBX Operations (pbx/)

Provides call control functions that publish to the EventBus:
- `OriginateAndPark()`: Initiate outbound call
- `OriginateToRobot()`: Call customer, then bridge to robot
- `HangupCall()`, `Transfer()`, `Playback()`: Session control

These send Lua scripts to FreeSWITCH via bgapi for async execution with promise-based responses.

### HTTP Server (modules/server/http_server/)

Built on Gin framework with CORS enabled. Routes register via `routers` map and `registers` slice. Grouped under `/api` prefix. Default admin user (admin/admin123) created on first run.

### Router (modules/server/router/)

Subscribes to `TOPIC_SESSION_ANSWER` to handle inbound/outbound call routing. For inbound: queries `TenantNumber` to find the robot, then routes based on action type. For outbound: retrieves Task from memory and routes to configured robot.

### Storage Layer (store/)

Pluggable store interface supporting different backends. Currently uses file-based GORM storage (`.db` directory). Manages entities: User (bcrypt password hashing), Tenant, TenantNumber (maps phone numbers to tenants/robots), Robot (AI/vendor configuration).

### Configuration (kcfg/)

Custom config format parsed from `config.kcfg`. Access via `cfg.Child("path").GetString()` or `GetInt()`. Key sections: freeswitch (ESL connection), backend (HTTP callbacks), http (server binding), jwt (auth secrets), llm (AI model settings).

### Lua Scripts (robots/, scripts/)

Used by pbx module for call flows:
- `do_originate_and_park.lua`: Bridge caller to parking lot
- `do_originate_to_robot.lua`: Bridge customer to AI/vendor robot

## Key Patterns

**Adding a new HTTP endpoint**: Create handler function, add to `modules/server/http_server/router.go` in `routers` map or `registers` slice.

**Adding a new FreeSWITCH event handler**: Add `esl.On("EVENT_NAME", callback)` in `esl_notify.Init()`, then publish to EventBus.

**Adding a new event type**: Define struct in `event/event.go`, add topic constant, publish with `GetDefaultBus().Publish()`.

**Adding a new PBX operation**: Create function in `pbx/*.go` that constructs `PBXCommand` and calls `GetDefaultBus().Request(TOPIC_SEND_API_WITH_PROMISE, cmd)`.

## Data Flow: Outbound Call to Robot

1. HTTP API receives originate request
2. Router creates Task in datacenter
3. `pbx.OriginateToRobot()` sends Lua script via ESL bgapi
4. FreeSWITCH calls customer phone
5. Customer answers → CHANNEL_ANSWER event
6. Router receives TOPIC_SESSION_ANSWER, routes to robot
7. PBX bridges customer to robot session
8. ASR results flow back via robot::asr custom event
