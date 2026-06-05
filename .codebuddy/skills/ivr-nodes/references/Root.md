# Root — 根节点

IVR 流程的入口节点，不执行任何操作，仅作为流程起点。

## 节点定义（来源: `skill.lua`）

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

## 使用样例（来源: `demo.lua`）

```lua
local root = Root:new()

-- 连接到第一个节点
root:connect(node_welcom_press_1)

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
