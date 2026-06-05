# Playback — 纯语音播放

仅播放语音文件，不做任何交互。适合提示语、等待音、结束语。

## 节点定义（来源: `skill.lua`）

```lua
local Playback = {}
Playback.__index = Playback
function Playback:new(file)
    local self = setmetatable({}, Playback)
    self.file = file
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.next_node = nil
    return self
end

function Playback:do_action()
    engine:playback(self.file)
    self.outputs = self.parent_node.outputs
    return self.next_node
end

function Playback:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end
```

## 构造参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `file` | string | 语音文件路径 |

## 方法

| 方法 | 说明 |
|------|------|
| `connect(node)` | 播放后的下一个节点（传 `nil` 则终止流程） |
| `do_action()` | 播放语音，返回 next_node |

## 输出

不产生新输出，`self.outputs` 继承自 `self.parent_node.outputs`。

## 使用样例（来源: `demo.lua`）

### 播放等待提示后继续

```lua
-- "正在处理中，请稍候..."
local node_payback_voice = Playback:new(menu_3)

-- 播放后进入校验
node_payback_voice:connect(node_check_6_digits)
```

### 播放成功音后终止

```lua
-- "校验通过"
local node_check_success = Playback:new(menu_3_success)

-- 播放后终止（next_node 保持 nil）
node_check_success:connect(nil)
```

### 播放失败音后终止

```lua
-- "校验失败，再见"
local node_final_fail = Playback:new(menu_4_failed)

node_final_fail:connect(nil)
```

### 完整流程中的位置

```lua
-- 输入成功 → 等待提示 → 后端校验
node_recv_6_digits:success_connect(node_payback_voice)   -- 输入完成
node_payback_voice:connect(node_check_6_digits)           -- 播放后校验

-- 校验通过 → 成功音 → 终止
node_ifelse:if_connect(..., node_check_success)
node_check_success:connect(nil)
```

## 典型场景

| 场景 | 示例 | connect |
|------|------|---------|
| 等待提示 | "正在处理，请稍候" | `connect(next_node)` |
| 成功通知 | "操作成功" | `connect(nil)` 终止 |
| 失败通知 | "操作失败，再见" | `connect(nil)` 终止 |
| 过渡语音 | 在输入和校验之间 | `connect(check_node)` |
