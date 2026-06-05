# PlayAndGetDigits — 播放语音并接收多位输入

播放语音文件，等待用户输入多位数字（如密码、编号）。

## 节点定义（来源: `skill.lua`）

```lua
local PlayAndGetDigits = {}
PlayAndGetDigits.__index = PlayAndGetDigits
function PlayAndGetDigits:new(file, hope_len, timeout)
    local self = setmetatable({}, PlayAndGetDigits)
    self.file = file
    self.hope_len = hope_len
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function PlayAndGetDigits:do_action()
    local get_digits = engine:play_and_get_digits(self.file, self.hope_len, self.timeout)
    self.outputs = self.parent_node.outputs
    if #get_digits == 0 then
        return self.fail_node
    end
    table.insert(self.outputs, {result = get_digits})
    return self.success_node
end

function PlayAndGetDigits:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function PlayAndGetDigits:fail_connect(node)
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
| `file` | string | 语音文件路径 |
| `hope_len` | number | 期望的数字位数 |
| `timeout` | number | 超时时间(毫秒) |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | 输入成功后的下一个节点 |
| `fail_connect(node)` | 超时/无输入后的下一个节点 |
| `do_action()` | 调用 `play_and_get_digits`，返回 success_node 或 fail_node |

## 输出

`self.outputs` 追加 `{result = get_digits}`（用户输入的数字字符串）

## 使用样例（来源: `demo.lua`）

### 普通输入

```lua
local node_recv_6_digits = PlayAndGetDigits:new(menu_2, 6, 10000)

node_recv_6_digits:success_connect(node_payback_voice)  -- 输入成功 → 播放等待音
node_recv_6_digits:fail_connect(nil)                     -- 无输入 → 挂断
```

### 循环内的重新输入

```lua
-- 校验失败后的重试输入
local node_retry_input = PlayAndGetDigits:new(menu_3_failed, 6, 10000)

node_retry_input:success_connect(node_retry_check)       -- 输入成功 → 重新校验
node_retry_input:fail_connect(node_retry_loop)            -- 无输入 → 回到 Loop（减少次数）
```

## 典型搭配

```lua
-- PlayAndGetDigits → Playback(等待音) → PlayAndRequestPost(校验) → IfElse(判断)
local input  = PlayAndGetDigits:new(input_voice, 6, 10000)
local wait   = Playback:new("请稍候...")
local check  = PlayAndRequestPost:new("local_stream://moh", check_url, body, 10000)
local ifelse = IfElse:new()

input:success_connect(wait)
wait:connect(check)
check:success_connect(ifelse)
```
