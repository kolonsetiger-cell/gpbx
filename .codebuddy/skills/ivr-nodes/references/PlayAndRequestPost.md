# PlayAndRequestPost — 播放等待音并发送 POST 请求

播放等待音（如音乐），同时向后端发送 JSON POST 请求，等待响应后继续。

## 节点定义（来源: `ivrs/skill.lua`）

```lua
local PlayAndRequestPost = {}
PlayAndRequestPost.__index = PlayAndRequestPost

function PlayAndRequestPost:new(file, url, body, timeout)
    local self = setmetatable({}, PlayAndRequestPost)
    self.file     = file
    self.url      = url
    self.body     = body or {}
    self.timeout  = timeout
    self.parent_node = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings    = {}  -- bind_node_output 注册的回调函数列表
    return self
end

function PlayAndRequestPost:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function PlayAndRequestPost:do_action()
    -- 合并前面节点的输出到 body
    for _, binding in ipairs(self.bindings) do
        local body_part = binding(self)
        if body_part ~= nil then
            self.body = deepMerge(self.body, body_part)
        end
    end

    local response, err = engine:play_and_request_post(self.file, self.url, self.body, self.timeout)
    if err ~= nil then
        return self.fail_node
    end
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    table.insert(self.outputs, response)
    return self.success_node
end

function PlayAndRequestPost:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function PlayAndRequestPost:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 构造参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `file` | string | 等待音文件（通常 `"local_stream://moh"` 音乐保持） |
| `url` | string | POST 请求的 URL |
| `body` | table | 请求体（JSON 对象，默认 `{}`） |
| `timeout` | number | 超时时间(毫秒) |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | `err == nil` 时的后续节点 |
| `fail_connect(node)` | `err ~= nil` 时的后续节点 |
| `bind_node_output(func_bind)` | 绑定前驱节点的输出，动态注入到 `self.body` |
| `do_action()` | 执行 bindings 合并 → 调用 `play_and_request_post`，返回 success_node 或 fail_node |

## bindings 机制

`bind_node_output` 允许在发送请求前，将上游节点的输出动态注入到请求体中：

```lua
local check = PlayAndRequestPost:new("local_stream://moh", check_url, {session_id = engine:get_uuid()}, 10000)

-- 绑定：将前驱节点的 output 注入到 body
check:bind_node_output(function(self)
    local input_node = self.parent_node  -- 前驱节点（如 PlayAndGetDigits）
    return {input_value = input_node.output}
end)
```

- 每个 binding 函数接收当前节点 `self`，返回一个 table
- 返回值通过 `deepMerge` 合并到 `self.body`
- 返回 `nil` 则不合并

## 输出

`self.outputs` 追加 `response`（服务端响应数据，通常含 `.code` 字段）

## 使用样例（来源: `ivrs/demo.lua`）

### 首次校验

```lua
local node_check_6_digits = PlayAndRequestPost:new(
    "local_stream://moh",                    -- 等待音
    check_url,                                -- 校验接口
    {session_id = engine:get_uuid()},         -- 请求体
    10000                                     -- 超时
)

node_check_6_digits:success_connect(node_6_digits_ifelse_check)  -- 请求成功 → 判断结果
node_check_6_digits:fail_connect(nil)                             -- 网络错误 → 挂断
```

### 带 bindings 的校验

```lua
local node_check = PlayAndRequestPost:new(
    "local_stream://moh",
    check_url,
    {session_id = engine:get_uuid()},  -- 基础 body
    10000
)

-- 将前驱节点的输入值注入到 body
node_check:bind_node_output(function(self)
    local input_node = self.parent_node  -- PlayAndGetDigits / PlayAndGetDigitsWithEnd
    return {input = input_node.output}   -- 合并到 self.body
end)

node_check:success_connect(node_ifelse)
node_check:fail_connect(nil)
```

### 循环内重新校验

```lua
local node_retry_check = PlayAndRequestPost:new(
    "local_stream://moh",
    check_url,
    {session_id = engine:get_uuid()},
    10000
)

node_retry_check:success_connect(node_retry_ifelse)   -- 请求成功 → 判断结果
node_retry_check:fail_connect(node_retry_loop)         -- 网络错误 → 回到 Loop
```

### 判断响应结果（配合 IfElse）

```lua
local ifelse = IfElse:new()
ifelse:if_connect(function(self)
    local response = self.outputs[#self.outputs]
    local code = response.code
    if code ~= 200 then
        return false
    end
    return true
end, node_success)        -- code == 200 → 进入下一阶段
ifelse:else_connect(node_retry_loop)  -- code != 200 → 重试
```

## 与 HttpPost 的区别

| | PlayAndRequestPost | HttpPost |
|------|------|------|
| 是否播放语音 | 是（等待音） | 否 |
| 引擎调用 | `play_and_request_post` | `post_json` |
| 成功条件 | `err == nil` | `code == 200 && err == nil` |
| 返回值 | `response, err` | `code, response, err` |
| 典型场景 | 校验等待（有等待音） | 通知、上报（无感知） |

## 典型流程

```
PlayAndGetDigits(输入) → Playback(等待提示) → PlayAndRequestPost(校验) → IfElse(判断)
```
