# Root — 根节点

机器人流程的入口节点，不执行任何操作，仅作为流程起点。

## 节点定义（来源: `robots/skill.lua`）

```lua
local Root = {}
Root.__index = Root

function Root:new()
    local self = setmetatable({}, Root)
    self.outputs = {}
    self.next_node = nil
    self.parent_node = nil
    self.error = nil
    return self
end

function Root:do_action()
    return self.next_node
end

function Root:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return Root
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `outputs` | table | 空输出容器，子节点继承 |
| `next_node` | node/nil | 第一个要执行的节点 |
| `parent_node` | nil | 始终为 nil |

## 方法

| 方法 | 说明 |
|------|------|
| `connect(node)` | 设置第一个要执行的节点 |
| `do_action()` | 返回 `next_node` |

## 使用样例（来源: `robots/demo.lua`）

```lua
local skill = require("robots/skill")
local Root = skill.Root
local Loop = skill.Loop

local root = Root:new()
local loop = Loop:new(-1)

root:connect(loop)

-- 主循环
local node = root
while engine:is_ok() do
    if node == nil then
        break
    end
    node = node:do_action()
end
```

Root 总是第一个被创建，也是主循环的起点。`do_action()` 只是简单地返回 `next_node`，不执行任何业务逻辑。
