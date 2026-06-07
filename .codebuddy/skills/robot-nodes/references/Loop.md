# Loop — 循环节点

循环执行子节点，支持无限循环和有限次数循环。

## 节点定义（来源: `robots/skill.lua`）

```lua
local Loop = {}
Loop.__index = Loop

function Loop:new(loop_count)
    local self = setmetatable({}, Loop)
    self.parent_node  = nil
    self.outputs      = nil
    self.error        = nil
    self.next_node    = nil
    self.fail_node    = nil
    self.loop_count   = loop_count
    return self
end

function Loop:do_action()
    if self.loop_count == 0 then
        return self.fail_node
    end
    if self.loop_count == -1 then
        return self.next_node
    end
    self.loop_count = self.loop_count - 1
    return self.next_node
end

function Loop:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function Loop:fail_connect(node)
    self.fail_node = node
    return self
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `loop_count` | number | 剩余循环次数：`-1`=无限，`0`=退出，`>0`=有限次数 |
| `next_node` | node/nil | 循环体节点（每次循环执行的节点）|
| `fail_node` | node/nil | 循环退出后的节点（`loop_count == 0` 时）|

## 方法

| 方法 | 说明 |
|------|------|
| `connect(node)` | 设置循环体节点 |
| `fail_connect(node)` | 设置循环退出节点 |
| `do_action()` | 判断循环次数，返回对应节点 |

## 使用样例（来源: `robots/demo.lua`）

### 无限循环（最常用）

```lua
local Loop = skill.Loop

-- 无限循环：持续等待用户输入
local loop = Loop:new(-1)

loop:connect(say_and_detect)  -- 循环体：播放提示并识别
loop:fail_connect(nil)         -- 无限循环，不会触发 fail
```

### 有限次数循环（重试）

```lua
-- 最多重试 3 次
local retry_loop = Loop:new(3)

retry_loop:connect(say_and_detect)   -- 循环体
retry_loop:fail_connect(hangup_node) -- 3 次失败后挂断
```

## 典型搭配

| 场景 | loop_count | 说明 |
|------|------------|------|
| 主循环（等待用户输入）| `-1` | 无限循环，直到用户说"退出" |
| 重试有限次数 | `>0` | 如 3 次识别失败后挂断 |
| 单次执行 | `1` | 等价于执行一次后退出 |
