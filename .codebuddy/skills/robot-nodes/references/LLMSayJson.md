# LLMSayJson — LLM 调用（JSON 响应）

将前面节点的输出通过 bindings 拼接为 context，调用 LLM 并解析返回的 JSON 结果。

## 节点定义（来源: `robots/skill.lua`）

```lua
local LLMSayJson = {}
LLMSayJson.__index = LLMSayJson

function LLMSayJson:new(prompt, timeout)
    local self = setmetatable({}, LLMSayJson)
    self.prompt   = prompt
    self.timeout  = timeout
    self.parent_node  = nil
    self.outputs      = nil
    self.output       = nil
    self.error        = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings     = {}
    return self
end

function LLMSayJson:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function LLMSayJson:do_action()
    local text = ""
    for _, binding in ipairs(self.bindings) do
        local body = binding(self)
        if body ~= nil then
            text = text .. body
        end
    end
    local response, err = engine:llm_say_json(self.prompt, text, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function LLMSayJson:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function LLMSayJson:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `prompt` | string | 系统提示词（定义 LLM 角色和输出格式）|
| `timeout` | number | LLM 调用超时（毫秒）|
| `output` | table | LLM 返回的 JSON 对象（已解析）|
| `bindings` | table | 回调函数列表，用于拼接上下文 |
| `success_node` | node/nil | 调用成功后的下一个节点 |
| `fail_node` | node/nil | 调用失败后的下一个节点 |

## 方法

| 方法 | 说明 |
|------|------|
| `bind_node_output(func)` | 注册回调函数，将其他节点的输出拼接到 LLM 上下文 |
| `success_connect(node)` | 设置调用成功分支 |
| `fail_connect(node)` | 设置调用失败分支 |
| `do_action()` | 拼接上下文，调用 LLM，返回对应分支节点 |

## 引擎 API

| 方法 | 说明 |
|------|------|
| `engine:llm_say_json(prompt, text, timeout)` | 调用 LLM 并解析 JSON 响应，返回 `(response, err)` |

## 使用样例（来源: `robots/demo.lua`）

```lua
local LLMSayJson = skill.LLMSayJson

local prompt = [[
你是专业智能话务台，只输出标准JSON，无任何多余内容。

### 可选意图
- tell_joke   : 讲笑话
- tell_story  : 讲故事
- bye         : 退出
- other       : 其他
]]

local llm_say = LLMSayJson:new(prompt, 5000)

-- 绑定 SayAndDetect 的输出作为 LLM 输入
llm_say:bind_node_output(function(node)
    return say_and_detect.output  -- 传入 ASR 识别结果
end)

llm_say:success_connect(if_else)  -- LLM 成功 → 条件判断
llm_say:fail_connect(loop)       -- LLM 失败 → 重新循环
```

## 下游访问 LLM 输出

```lua
-- 在 IfElse 条件函数中访问 LLM 输出
if_else:if_connect(function(node)
    if llm_say.output and llm_say.output.intent == "tell_joke" then
        return true
    end
    return false
end, say_joke)
```

## 与 LLMSayRaw 的区别

| | LLMSayJson | LLMSayRaw |
|--|-------------|------------|
| LLM 响应格式 | 必须是 JSON（自动解析为 table）| 原始文本 |
| `output` 类型 | table | string |
| 典型用途 | 意图识别、结构化数据提取 | 自由对话、文本生成 |
