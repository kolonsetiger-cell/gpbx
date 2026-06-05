# IfElse — 条件判断节点

根据条件函数判断结果，路由到不同分支（支持 if/elseif/else）。

## 节点定义（来源: `skill.lua`）

```lua
local IfElse = {}
IfElse.__index = IfElse
function IfElse:new()
    local self = setmetatable({}, IfElse)
    self.condition = nil
    self.true_node = nil
    self.elseif_node = {}
    self.else_node = nil
    self.inputs = nil
    return self
end

function IfElse:do_action()
    self.outputs = self.parent_node.outputs
    if self.condition(self) then
        return self.true_node
    end
    for i, v in ipairs(self.elseif_node) do
        if v.condition(self) then
            return v.node
        end
    end
    return self.else_node
end

function IfElse:if_connect(condition, node)
    self.true_node = node
    self.condition = condition
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function IfElse:else_connect(node)
    self.else_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function IfElse:ifelse_connect(condition, node)
    self.else_node = node
    node.parent_node = self
    table.insert(self.elseif_node, {condition = condition, node = node})
    return self
end
```

## 方法

| 方法 | 说明 |
|------|------|
| `if_connect(condition_fn, node)` | 设置 if 条件分支 |
| `ifelse_connect(condition_fn, node)` | 追加 elseif 条件分支 |
| `else_connect(node)` | 设置 else 分支（所有条件不满足时） |
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

## 使用样例（来源: `demo.lua`）

### 基本 if/else — 校验结果判断

```lua
local node_6_digits_ifelse_check = IfElse:new()

node_6_digits_ifelse_check:if_connect(function(self)
    local response = self.outputs[#self.outputs]
    local code = response.code
    if code ~= 200 then
        return false
    end
    return true
end, node_6_digits_check_success)          -- 校验通过 → 播放成功音

node_6_digits_ifelse_check:else_connect(node_6_digits_check_failed_loop)  -- 失败 → 进入重试
```

### if/elseif/else — 多分支按键判断

```lua
local ifelse = IfElse:new()

ifelse:if_connect(function(self)
    return self.outputs[#self.outputs][1] == "1"    -- 按1
end, menu_1_node)

ifelse:ifelse_connect(function(self)
    return self.outputs[#self.outputs][1] == "2"    -- 按2
end, menu_2_node)

ifelse:else_connect(hangup_node)                     -- 其他 → 挂断
```

### 循环内的判断

```lua
local node_retry_ifelse = IfElse:new()

node_retry_ifelse:if_connect(function(self)
    local response = self.outputs[#self.outputs]
    return response.code == 200
end, next_stage_node)           -- 成功 → 离开循环，进入下一阶段

node_retry_ifelse:else_connect(node_retry_loop)  -- 失败 → 继续循环
```

## 典型搭配

| 上游节点 | 判断内容 | 典型分支 |
|----------|----------|----------|
| `PlayAndRequestPost` | `response.code == 200` | 成功 → 下一阶段 / 失败 → Loop |
| `PlayAndGetDigit` | `outputs[#outputs][1] == "1"` | 按1 → A / 按2 → B / 其他 → 挂断 |
| `HttpPost` | `response.code == 200` | 成功/失败都继续（通知类） |
