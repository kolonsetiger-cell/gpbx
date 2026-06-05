# Loop — 循环重试节点

实现带重试次数的循环逻辑。每次执行递减计数，计数归零时走失败出口。

## 节点定义（来源: `skill.lua`）

```lua
local Loop = {}
Loop.__index = Loop
function Loop:new(loop_count)
    local self = setmetatable({}, Loop)
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.next_node = nil
    self.fail_node = nil
    self.loop_count = loop_count
    return self
end

function Loop:do_action()
    if self.loop_count == 0 then
        return self.fail_node
    end
    self.loop_count = self.loop_count - 1
    return self.next_node
end

function Loop:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self.parent_node
    return self
end

function Loop:fail_connect(node)
    self.fail_node = node
    return self
end
```

## 构造参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `loop_count` | number | 最大循环/重试次数 |

## 方法

| 方法 | 说明 |
|------|------|
| `connect(node)` | 循环体节点（每次循环执行的第一个节点） |
| `fail_connect(node)` | 循环次数耗尽后的出口 |
| `do_action()` | 递减计数，返回循环体或 fail_node |

## 输出

不产生新输出，`self.outputs` 继承自 `self.parent_node.outputs`。

## 核心机制

Loop 的关键特性：**循环体内的失败出口要连回 Loop**，成功出口要离开 Loop。

```
Loop(N) ──→ [循环体: 输入 → 校验 → 判断]
  ↑              ├─ 失败 → Loop (继续)
  │              └─ 成功 → 下一阶段 (离开)
  └── 耗尽 → fail_node (最终失败)
```

## 使用样例（来源: `demo.lua`）

### 完整重试校验流程

```lua
-- 创建重试循环（最多3次）
local node_retry_loop = Loop:new(3)

-- 循环内节点
local node_retry_input  = PlayAndGetDigits:new(menu_3_failed, 6, 10000)
local node_retry_check  = PlayAndRequestPost:new("local_stream://moh", check_url, {session_id = engine:get_uuid()}, 10000)
local node_retry_ifelse = IfElse:new()

-- 连接 Loop
node_retry_loop:connect(node_retry_input)           -- 循环体入口
node_retry_loop:fail_connect(nil)                   -- 3次耗尽 → 挂断

-- 循环体内部连接
node_retry_input:success_connect(node_retry_check)   -- 输入成功 → 校验
node_retry_input:fail_connect(node_retry_loop)        -- 无输入 → 回 Loop（计次-1）

node_retry_check:success_connect(node_retry_ifelse)   -- 请求成功 → 判断
node_retry_check:fail_connect(node_retry_loop)         -- 网络错误 → 回 Loop

-- 判断结果：成功离开，失败继续
node_retry_ifelse:if_connect(function(self)
    local response = self.outputs[#self.outputs]
    return response.code == 200
end, next_stage_node)        -- ✅ 成功 → 离开循环，进入下一阶段

node_retry_ifelse:else_connect(node_retry_loop)  -- ❌ 失败 → 回 Loop
```

### 与主流程的连接

```lua
-- 首次校验失败 → 进入重试
node_first_ifelse:else_connect(node_retry_loop)

-- 循环内校验成功 → 跳到下一阶段（跳过 Loop 的 fail_node）
node_retry_ifelse:if_connect(function(self)
    return self.outputs[#self.outputs].code == 200
end, node_recv_4_digits)  -- 直接跳到第4步
```

## 关键规则

1. **成功离开**：循环内 IfElse 的成功出口必须指向循环外的节点（如下一阶段）
2. **失败继续**：循环内所有失败出口（输入失败、请求失败、校验失败）都连回 Loop
3. **耗尽处理**：`fail_connect` 设置次数耗尽后的出口（通常 `nil` 挂断）
4. **parent_node**：`Loop:connect` 使用 `self.parent_node` 设置循环体的父节点，确保 outputs 正确继承
