# IVR Node 技能索引

基于 `ivrs/skill.lua`（节点定义库）和 `ivrs/demo.lua`（使用样例）生成的技能参考文档。

## 节点列表

| 节点 | 文件 | 用途 | 引擎调用 |
|------|------|------|----------|
| Root | [Root.md](./Root.md) | 流程入口 | 无 |
| PlayAndGetDigit | [PlayAndGetDigit.md](./PlayAndGetDigit.md) | 播放+单键输入 | `play_and_get_digit` |
| PlayAndGetDigits | [PlayAndGetDigits.md](./PlayAndGetDigits.md) | 播放+多位输入 | `play_and_get_digits` |
| PlayAndGetDigitsWithEnd | [PlayAndGetDigitsWithEnd.md](./PlayAndGetDigitsWithEnd.md) | 播放+多位输入(结束键) | `play_and_get_digits_with_end` |
| PlayAndRequestPost | [PlayAndRequestPost.md](./PlayAndRequestPost.md) | 播放等待音+POST | `play_and_request_post` |
| IfElse | [IfElse.md](./IfElse.md) | 条件判断 | 无 |
| HttpPost | [HttpPost.md](./HttpPost.md) | 纯POST请求 | `post_json` |
| Playback | [Playback.md](./Playback.md) | 纯语音播放 | `playback` |
| Loop | [Loop.md](./Loop.md) | 循环重试 | 无 |

## 快速组合表

| 需求 | 节点组合 |
|------|----------|
| 播放后挂断 | `Playback → nil` |
| 单键菜单 | `PlayAndGetDigit → IfElse → ...` |
| 多位输入校验 | `PlayAndGetDigits → Playback → PlayAndRequestPost → IfElse` |
| 不定长输入校验 | `PlayAndGetDigitsWithEnd → PlayAndRequestPost(bind_node_output) → IfElse` |
| 校验+重试 | 上面 + `Loop` |
| 无声通知 | `HttpPost`（成功失败都继续） |
| 等待+校验 | `PlayAndRequestPost → IfElse` |
| 动态注入上游输出 | `PlayAndRequestPost:bind_node_output` / `HttpPost:bind_node_output` |

## 完整流程示例（来源: `ivrs/demo.lua`）

```
Root
 └─ PlayAndGetDigit (按1)
      ├─ fail → nil (挂断)
      └─ success → HttpPost (通知)
                     ├─ fail → PlayAndGetDigits (6位输入)
                     └─ success → PlayAndGetDigits (6位输入)
                                    ├─ fail → nil
                                    └─ success → Playback (等待音)
                                                   └─ PlayAndRequestPost (校验6位)
                                                        ├─ fail → nil
                                                        └─ success → IfElse
                                                                       ├─ 200 → Playback(成功)
                                                                       │          └─ PlayAndGetDigits (4位输入)
                                                                       │               └─ PlayAndRequestPost (校验4位)
                                                                       │                    └─ IfElse
                                                                       │                         ├─ 200 → Playback(结束)
                                                                       │                         └─ 非200 → Loop(3)
                                                                       └─ 非200 → Loop(3)
                                                                                    ├─ PlayAndGetDigits (重输6位)
                                                                                    │    └─ PlayAndRequestPost (重校6位)
                                                                                    │         └─ IfElse
                                                                                    │              ├─ 200 → 下一阶段
                                                                                    │              └─ 非200 → Loop
                                                                                    └─ 耗尽 → nil
```

## 节点输出继承链

每个节点在 `do_action()` 中执行 `self.outputs = self.parent_node and self.parent_node.outputs or {}`，然后 `table.insert` 自己的结果。

下游节点通过 `self.outputs[#self.outputs]` 获取最后一个输出：

```lua
-- 获取 PlayAndRequestPost 的响应
local response = self.outputs[#self.outputs]  -- {code = 200, ...}

-- 获取 PlayAndGetDigit 的按键
local digit = self.outputs[#self.outputs][1]  -- "1"

-- 获取 PlayAndGetDigits 的输入
local digits = self.outputs[#self.outputs].result  -- "123456"
```

## bind_node_output 绑定模式

`PlayAndRequestPost` 和 `HttpPost` 支持 `bind_node_output`，可将上游节点的 `self.output` 字段动态注入到请求体：

```lua
-- 上游节点（如 PlayAndGetDigitsWithEnd）执行后 self.output = 用户输入
-- 下游节点通过 bind_node_output 访问
check_node:bind_node_output(function(self)
    local input_node = self.parent_node  -- 访问前驱节点
    return {account = input_node.output} -- 注入到 body
end)
```

此模式替代了手动在构造时硬编码 body 的方式，使请求体可以动态包含用户输入。
