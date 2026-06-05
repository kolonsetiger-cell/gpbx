# PlayAndGetDigit — 播放语音并接收单键输入

播放语音文件，等待用户按下一个 DTMF 键。

## 节点定义（来源: `skill.lua`）

```lua
local PlayAndGetDigit = {}
PlayAndGetDigit.__index = PlayAndGetDigit
function PlayAndGetDigit:new(file, hope_digit, timeout)
    local self = setmetatable({}, PlayAndGetDigit)
    self.file = file
    self.hope_digit = hope_digit
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function PlayAndGetDigit:do_action()
    local get_digit = engine:play_and_get_digit(self.file, self.hope_digit, self.timeout)
    self.outputs = self.parent_node.outputs
    if #get_digit == 0 then
        return self.fail_node
    end
    table.insert(self.outputs, {get_digit})
    return self.success_node
end

function PlayAndGetDigit:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function PlayAndGetDigit:fail_connect(node)
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
| `hope_digit` | string | 期望的按键（如 `"1"`） |
| `timeout` | number | 超时时间(毫秒) |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | 按键正确后的下一个节点 |
| `fail_connect(node)` | 超时/无输入后的下一个节点 |
| `do_action()` | 调用 `play_and_get_digit`，返回 success_node 或 fail_node |

## 输出

`self.outputs` 追加 `{get_digit}`（用户按键值）

## 使用样例（来源: `demo.lua`）

### 节点实例化

```lua
local node_welcom_press_1 = PlayAndGetDigit:new(menu_1, "1", 10000)
```

### 节点连接

```lua
root:connect(node_welcom_press_1)
node_welcom_press_1:success_connect(node_notify_digit_1)  -- 按1 → 通知服务
node_welcom_press_1:fail_connect(nil)                     -- 超时 → 挂断
```

### 条件判断按键值

```lua
-- 配合 IfElse 判断按键
local ifelse = IfElse:new()
ifelse:if_connect(function(self)
    return self.outputs[#self.outputs][1] == "1"
end, menu_1_node)
ifelse:ifelse_connect(function(self)
    return self.outputs[#self.outputs][1] == "2"
end, menu_2_node)
ifelse:else_connect(hangup_node)
```

## 与 PlayAndGetDigits 的区别

| | PlayAndGetDigit | PlayAndGetDigits |
|------|------|------|
| 引擎调用 | `play_and_get_digit` | `play_and_get_digits` |
| 输入方式 | 单次按键 | 多位数字输入 |
| 输出格式 | `{get_digit}` | `{result = get_digits}` |
| 典型场景 | "请按1" | "请输入6位密码" |
