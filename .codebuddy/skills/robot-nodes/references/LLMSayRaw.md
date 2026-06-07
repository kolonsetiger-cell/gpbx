# LLMSayRaw — LLM 调用（原始响应）

将前面节点的输出通过 bindings 拼接为 context，调用 LLM 并返回原始文本（非 JSON）。

## 节点定义（来源: `robots/skill.lua`）

```lua
local LLMSayRaw = {}
LLMSayRaw.__index = LLMSayRaw

function LLMSayRaw:new(prompt, timeout)
    local self = setmetatable({}, LLMSayRaw)
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

function LLMSayRaw:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function LLMSayRaw:do_action()
    local text = ""
    for _, binding in ipairs(self.bindings) do
        local body = binding(self)
        if body ~= nil then
            text = text .. body
        end
    end
    local response, err = engine:llm_say_raw(self.prompt, text, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function LLMSayRaw:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function LLMSayRaw:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `prompt` | string | 系统提示词 |
| `timeout` | number | LLM 调用超时（毫秒）|
| `output` | string | LLM 返回的原始文本 |
| `bindings` | table | 回调函数列表，用于拼接上下文 |
| `success_node` | node/nil | 调用成功后的下一个节点 |
| `fail_node` | node/nil | 调用失败后的下一个节点 |

## 方法

| 方法 | 说明 |
|------|------|
| `bind_node_output(func)` | 注册回调函数，拼接上下文 |
| `success_connect(node)` | 设置调用成功分支 |
| `fail_connect(node)` | 设置调用失败分支 |
| `do_action()` | 拼接上下文，调用 LLM，返回对应分支节点 |

## 引擎 API

| 方法 | 说明 |
|------|------|
| `engine:llm_say_raw(prompt, text, timeout)` | 调用 LLM 返回原始文本，返回 `(response, err)` |

## 使用样例

```lua
local LLMSayRaw = skill.LLMSayRaw

-- 自由对话场景：将 ASR 结果直接传给 LLM，返回原始回复
local llm_raw = LLMSayRaw:new(
    "你是一个友好的客服助手，用简洁的中文回复用户。",
    8000)

llm_raw:bind_node_output(function(node)
    return say_and_detect.output  -- ASR 识别结果
end)

llm_raw:success_connect(say_reply)  -- 成功 → 播放 LLM 回复
llm_raw:fail_connect(loop)        -- 失败 → 重新循环
```

## 与 LLMSayJson 的区别

| | LLMSayJson | LLMSayRaw |
|--|-------------|------------|
| LLM 响应格式 | 必须是 JSON（自动解析为 table）| 原始文本 |
| `output` 类型 | table | string |
| 典型用途 | 意图识别、结构化数据提取 | 自由对话、文本生成 |
