# HttpPost — 纯 HTTP POST 请求

发送 JSON POST 请求，不播放语音。适合通知/上报场景。

## 节点定义（来源: `skill.lua`）

```lua
local HttpPost = {}
HttpPost.__index = HttpPost
function HttpPost:new(url, header, body, timeout)
    local self = setmetatable({}, HttpPost)
    self.url = url
    self.header = header
    self.body = body
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function HttpPost:do_action()
    local code, response, err = engine:post_json(self.url, {}, {session_id = engine:get_uuid()}, 10000)
    self.outputs = self.parent_node.outputs
    if code ~= 200 or err ~= nil then
        return self.fail_node
    end
    table.insert(self.outputs, {response})
    return self.success_node
end

function HttpPost:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function HttpPost:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end
```

## 构造参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `url` | string | POST 请求的 URL |
| `header` | table | 请求头（通常传 `{}`） |
| `body` | table | 请求体（JSON 对象） |
| `timeout` | number | 超时时间(毫秒) |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | HTTP 200 且无错误的后续节点 |
| `fail_connect(node)` | 非 200 或有错误的后续节点 |
| `do_action()` | 调用 `post_json`，返回 success_node 或 fail_node |

## 输出

`self.outputs` 追加 `{response}`（服务端响应数据）

## 使用样例（来源: `demo.lua`）

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

- `do_action()` 中硬编码了 `self.header` → `{}`，`self.body` → `{session_id = engine:get_uuid()}`，构造时的 `header` 和 `body` 参数实际上未在 `do_action` 中使用。如需自定义 body，建议使用 `PlayAndRequestPost`。
