# SaySync — 同步播放

播放 TTS 语音（等待播报完成），返回 ASR 识别结果。

## 节点定义（来源: `robots/skill.lua`）

```lua
local SaySync = {}
SaySync.__index = SaySync

function SaySync:new(text)
    local self = setmetatable({}, SaySync)
    self.text = text
    self.parent_node  = nil
    self.outputs      = nil
    self.output       = nil
    self.error        = nil
    self.success_node = nil
    self.fail_node    = nil
    return self
end

function SaySync:do_action()
    local response, err = engine:say_sync(self.text)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function SaySync:connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end
```

## 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `text` | string | TTS 播报文本 |
| `output` | string | ASR 识别结果 |
| `success_node` | node/nil | 成功后的下一个节点（通过 `connect` 设置）|
| `fail_node` | node/nil | 失败后的下一个节点 |

## 方法

| 方法 | 说明 |
|------|------|
| `connect(node)` | 设置成功分支（与 SayAndDetect:success_connect 接口一致）|
| `do_action()` | 执行 TTS+ASR，返回对应分支节点 |

## 引擎 API

| 方法 | 说明 |
|------|------|
| `engine:say_sync(text)` | 播放 TTS 并等待 ASR 输入，返回 `(response, err)` |

## 使用样例（来源: `robots/demo.lua`）

```lua
local SaySync = skill.SaySync

local say_joke = SaySync:new("大明捡了2分钱，但是钱存了")
local say_story = SaySync:new("小明捡了一分钱，但是钱没了。")

-- SaySync 通常作为流程的叶子节点（connect(nil) 或 fail_connect(nil) 结束流程）
say_joke:connect(nil)
```

## 与 SayAndDetect 的区别

| | SayAndDetect | SaySync |
|--|--------------|---------|
| 播报后行为 | 等待 ASR 输入 | 播报完成后返回 ASR 结果 |
| 设置成功分支 | `success_connect(node)` | `connect(node)` |
| 典型用途 | 交互式对话（先听用户说） | 播报结果后结束或继续 |
