# IfElse — 条件判断节点

根据条件函数判断结果，路由到不同分支（支持 if/elseif/else）。

## 节点定义（来源: `robots/skill.lua`）

```lua
local IfElse = {}
IfElse.__index = IfElse

function IfElse:new()
    local self = setmetatable({}, IfElse)
    self.condition    = nil
    self.true_node    = nil
    self.elseif_node  = {}
    self.else_node    = nil
    self.inputs       = nil
    return self
end

function IfElse:do_action()
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if self.condition and self.condition(self) then
        return self.true_node
    end
    for _, v in ipairs(self.elseif_node) do
        if v.condition and v.condition(self) then
            return v.node
        end
    end
    return self.else_node
end

function IfElse:if_connect(condition, node)
    self.true_node = node
    self.condition = condition
    if node == nil then return self end
    node.parent_node = self
    return self
end

function IfElse:else_connect(node)
    self.else_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function IfElse:ifelse_connect(condition, node)
    node.parent_node = self
    table.insert(self.elseif_node, { condition = condition, node = node })
    return self
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `condition` | function/nil | if 分支的条件函数 |
| `true_node` | node/nil | if 条件满足时的下一个节点 |
| `elseif_node` | table | elseif 分支列表，格式：`{ condition = func, node = node }` |
| `else_node` | node/nil | 所有条件不满足时的默认节点 |

## 方法

| 方法 | 说明 |
|------|------|
| `if_connect(condition_fn, node)` | 设置 if 条件分支 |
| `ifelse_connect(condition_fn, node)` | 追加 elseif 条件分支（可多次调用）|
| `else_connect(node)` | 设置 else 分支（所有条件不满足时）|
| `do_action()` | 依次判断条件，返回匹配的分支节点 |

## 条件函数签名

```lua
function(ifelse_node)  -- self = ifelse_node
    -- 通过 ifelse_node.outputs 访问上游输出
    -- 通过 ifelse_node.outputs[#ifelse_node.outputs] 获取最后一个输出
    return true   -- 走 true_node
    return false  -- 继续检查 elseif / else
end
```

## 使用样例（来源: `robots/demo.lua`）

### 基本 if/else — 意图路由

```lua
local IfElse = skill.IfElse
local LLMSayJson = skill.LLMSayJson

local if_else = IfElse:new()
local llm_say = LLMSayJson:new(prompt, 5000)

-- if: 讲笑话
if_else:if_connect(function(self)
    return llm_say.output and llm_say.output.intent == "tell_joke"
end, say_joke)

-- elseif: 讲故事
if_else:ifelse_connect(function(self)
    return llm_say.output and llm_say.output.intent == "tell_story"
end, say_story)

-- elseif: 退出
if_else:ifelse_connect(function(self)
    return llm_say.output and llm_say.output.intent == "bye"
end, nil)  -- nil → 结束流程

-- else: 重新循环
if_else:else_connect(loop)
```

## 典型搭配

| 上游节点 | 判断内容 | 典型分支 |
|----------|----------|----------|
| `LLMSayJson` | `output.intent == "xxx"` | 意图路由到不同 SaySync |
| `SayAndDetect` | `output == nil` | 识别失败 → Loop |
| 任意节点 | 自定义条件 | 动态路由 |
