# HttpPost — 纯 HTTP POST 请求

发送 JSON POST 请求，不播放语音。适合通知/上报场景。

## 节点定义（来源: `ivrs/skill.lua`）

```lua
local HttpPost = {}
HttpPost.__index = HttpPost

function HttpPost:new(url, header, body, timeout)
    local self = setmetatable({}, HttpPost)
    self.url      = url
    self.header   = header or {}
    self.body     = body or {}
    self.timeout  = timeout or 10000
    self.parent_node = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings     = {}  -- 修复原 bug：原代码缺少 bindings 初始化
    return self
end

function HttpPost:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function HttpPost:do_action()
    -- 合并前面节点的输出到 body
    for _, binding in ipairs(self.bindings) do
        local body_part = binding(self)
        if body_part ~= nil then
            self.body = deepMerge(self.body, body_part)
        end
    end

    local code, response, err = engine:post_json(self.url, self.header, self.body, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if code ~= 200 or err ~= nil then
        return self.fail_node
    end
    table.insert(self.outputs, { response })
    return self.success_node
end

function HttpPost:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function HttpPost:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 构造参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `url` | string | POST 请求的 URL |
| `header` | table | 请求头（默认 `{}`） |
| `body` | table | 请求体（JSON 对象，默认 `{}`） |
| `timeout` | number | 超时时间(毫秒)，默认 10000 |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | HTTP 200 且无错误的后续节点 |
| `fail_connect(node)` | 非 200 或有错误的后续节点 |
| `bind_node_output(func_bind)` | 绑定前驱节点的输出，动态注入到 `self.body` |
| `do_action()` | 执行 bindings 合并 → 调用 `post_json`，返回 success_node 或 fail_node |

## bindings 机制

与 `PlayAndRequestPost` 相同，`bind_node_output` 允许在发送请求前，将上游节点的输出动态注入到请求体中：

```lua
local notify = HttpPost:new(notify_url, {}, {session_id = engine:get_uuid()}, 10000)

notify:bind_node_output(function(self)
    local input_node = self.parent_node  -- 如 PlayAndGetDigit
    return {pressed_key = input_node.output}
end)
```

## 输出

`self.outputs` 追加 `{response}`（服务端响应数据）

## 使用样例（来源: `ivrs/demo.lua`）

### 通知上报

```lua
local node_notify_digit_1 = HttpPost:new(
    notify_url,                              -- 通知接口
    {},                                      -- header
    {session_id = engine:get_uuid()},        -- body
    10000                                    -- timeout
)

-- 成功失败都继续（通知不需要阻塞流程）
node_notify_digit_1:success_connect(node_recv_6_digits)
node_notify_digit_1:fail_connect(node_recv_6_digits)
```

### 带 bindings 的通知

```lua
local node_notify = HttpPost:new(
    notify_url, {}, {session_id = engine:get_uuid()}, 10000
)

-- 将按键值注入通知 body
node_notify:bind_node_output(function(self)
    local digit_node = self.parent_node  -- PlayAndGetDigit
    return {pressed_key = digit_node.output}
end)

node_notify:success_connect(node_next)
node_notify:fail_connect(node_next)
```

### 流程位置

在 demo.lua 中，HttpPost 用于按键后的通知上报，位于流程早期：

```
Root → PlayAndGetDigit(按1) → HttpPost(通知) → PlayAndGetDigits(输入)
                                   ↓
                         成功/失败都继续 → 不阻塞
```

## 与 PlayAndRequestPost 的区别

| | HttpPost | PlayAndRequestPost |
|------|------|------|
| 是否播放语音 | 否 | 是（等待音） |
| 引擎调用 | `post_json` | `play_and_request_post` |
| 成功条件 | `code == 200 && err == nil` | `err == nil` |
| 返回值 | `code, response, err` | `response, err` |
| 典型场景 | 通知、上报（无感知） | 校验等待（有等待音） |

## 注意事项

- `header` 参数支持自定义请求头，构造时传入即可（如 `{["Content-Type"] = "application/json"}`）
- 可通过 `bind_node_output` 将上游节点的输出动态合并到 `self.body` 中
- `timeout` 默认 10000 毫秒（10秒）
