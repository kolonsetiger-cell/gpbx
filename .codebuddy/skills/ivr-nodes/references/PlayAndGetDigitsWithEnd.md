# PlayAndGetDigitsWithEnd — 播放语音并接收多位输入（支持结束键）

播放语音文件，等待用户输入多位数字，用户按指定结束键（DTMF）时停止输入。与 `PlayAndGetDigits` 的区别在于可以指定一个 DTMF 键作为输入结束标志（如 `#`）。

## 节点定义（来源: `ivrs/skill.lua`）

```lua
local PlayAndGetDigitsWithEnd = {}
PlayAndGetDigitsWithEnd.__index = PlayAndGetDigitsWithEnd

function PlayAndGetDigitsWithEnd:new(file, hope_dtmf, timeout)
    local self = setmetatable({}, PlayAndGetDigitsWithEnd)
    self.file       = file
    self.hope_dtmf  = hope_dtmf
    self.timeout    = timeout
    self.parent_node  = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.output      = nil
    return self
end

function PlayAndGetDigitsWithEnd:do_action()
    local get_digits = engine:play_and_get_digits_with_end(self.file, self.hope_dtmf, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if #get_digits == 0 then
        return self.fail_node
    end
    self.output = get_digits
    table.insert(self.outputs, {result = get_digits})
    return self.success_node
end

function PlayAndGetDigitsWithEnd:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function PlayAndGetDigitsWithEnd:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 构造参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `file` | string | 语音文件路径 |
| `hope_dtmf` | string | 期望的结束 DTMF 键（如 `"#"`） |
| `timeout` | number | 超时时间(毫秒) |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | 输入成功后的下一个节点 |
| `fail_connect(node)` | 超时/无输入后的下一个节点 |
| `do_action()` | 调用 `play_and_get_digits_with_end`，返回 success_node 或 fail_node |

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `output` | string/nil | 本次输入的数字字符串，绑定（`bind_node_output`）时可直接访问 |

## 输出

`self.outputs` 追加 `{result = get_digits}`（用户输入的数字字符串）

## 使用样例

### 接收以 # 结束的输入

```lua
-- "请输入您的卡号，按#号结束"
local node_input_card = PlayAndGetDigitsWithEnd:new(card_input_voice, "#", 30000)

node_input_card:success_connect(node_check_card)  -- 输入成功 → 校验
node_input_card:fail_connect(nil)                  -- 超时 → 挂断
```

### 配合 bind_node_output 绑定输入值

```lua
local node_input = PlayAndGetDigitsWithEnd:new(voice_file, "#", 10000)

-- 下游 HttpPost/PlayAndRequestPost 绑定此节点的 output
node_check:bind_node_output(function(self)
    return {card_number = self.parent_node.output}
end)
```

## 与 PlayAndGetDigits 的区别

| | PlayAndGetDigits | PlayAndGetDigitsWithEnd |
|------|------|------|
| 引擎调用 | `play_and_get_digits` | `play_and_get_digits_with_end` |
| 输入结束方式 | 固定长度 `hope_len` | 结束键 `hope_dtmf` |
| 适用场景 | 固定位数输入（如6位密码） | 不定长输入（如卡号、账号） |
| 输出格式 | `{result = get_digits}` | `{result = get_digits}` |
| `self.output` | ✅ | ✅ |

## 典型搭配

```lua
-- PlayAndGetDigitsWithEnd → Playback(等待音) → PlayAndRequestPost(校验) → IfElse(判断)
local input  = PlayAndGetDigitsWithEnd:new(input_voice, "#", 30000)
local wait   = Playback:new("请稍候...")
local check  = PlayAndRequestPost:new("local_stream://moh", check_url, {}, 10000)

check:bind_node_output(function(self)
    return {account = input.output}  -- 将输入值绑定到请求体
end)

input:success_connect(wait)
wait:connect(check)
```
