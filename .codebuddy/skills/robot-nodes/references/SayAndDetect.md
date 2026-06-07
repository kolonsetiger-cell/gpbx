# SayAndDetect — 播放并识别

播放 TTS 提示语音，等待用户说话并通过 ASR 识别用户输入。

## 节点定义（来源: `robots/skill.lua`）

```lua
local SayAndDetect = {}
SayAndDetect.__index = SayAndDetect

function SayAndDetect:new(text, timeout)
    local self = setmetatable({}, SayAndDetect)
    self.text     = text     -- TTS 提示文本
    self.timeout  = timeout  -- ASR 等待超时（毫秒）
    self.parent_node = nil
    self.outputs     = nil
    self.output      = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    return self
end

function SayAndDetect:do_action()
    local response, err = engine:say_and_detect(self.text, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function SayAndDetect:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function SayAndDetect:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `text` | string | TTS 提示文本 |
| `timeout` | number | ASR 等待超时（毫秒） |
| `output` | string | ASR 识别结果 |
| `success_node` | node/nil | 识别成功后的下一个节点 |
| `fail_node` | node/nil | 识别失败/超时后的下一个节点 |

## 方法

| 方法 | 说明 |
|------|------|
| `success_connect(node)` | 设置识别成功分支 |
| `fail_connect(node)` | 设置识别失败分支 |
| `do_action()` | 执行 ASR，返回对应分支节点 |

## 引擎 API

| 方法 | 说明 |
|------|------|
| `engine:say_and_detect(text, timeout)` | 播放 TTS 并等待 ASR 输入，返回 `(response, err)` |

## 使用样例（来源: `robots/demo.lua`）

```lua
local SayAndDetect = skill.SayAndDetect

local say_and_detect = SayAndDetect:new(
    "欢迎使用我的机器人服务，我支持讲笑话，讲故事，告诉我您想做什么", 5000)

say_and_detect:success_connect(llm_say)    -- 识别成功 → LLM 处理
say_and_detect:fail_connect(loop)          -- 识别失败 → 重新循环
```

## 典型搭配

| 上游节点 | 下游节点 | 说明 |
|----------|----------|------|
| Root/Loop | SayAndDetect | 播放提示并等待用户输入 |
| SayAndDetect (success) | LLMSayJson | 将 ASR 结果传给 LLM |
| SayAndDetect (fail) | Loop | 识别失败，重新提示 |
