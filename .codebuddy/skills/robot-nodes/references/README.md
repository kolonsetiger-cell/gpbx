# Robot Node 技能索引

基于 `robots/skill.lua`（节点定义库）和 `robots/demo.lua`（使用样例）生成的技能参考文档。

## 节点列表

| 节点 | 文件 | 用途 | 引擎调用 |
|------|------|------|----------|
| Root | [Root.md](./Root.md) | 流程入口 | 无 |
| SayAndDetect | [SayAndDetect.md](./SayAndDetect.md) | 播放TTS+ASR识别 | `say_and_detect` |
| SaySync | [SaySync.md](./SaySync.md) | 同步播放TTS | `say_sync` |
| LLMSayJson | [LLMSayJson.md](./LLMSayJson.md) | LLM调用(JSON响应) | `llm_say_json` |
| LLMSayRaw | [LLMSayRaw.md](./LLMSayRaw.md) | LLM调用(原始响应) | `llm_say_raw` |
| IfElse | [IfElse.md](./IfElse.md) | 条件判断 | 无 |
| Loop | [Loop.md](./Loop.md) | 循环重试 | 无 |

## 快速组合表

| 需求 | 节点组合 |
|------|----------|
| 简单对话（识别意图+回复）| `Root → SayAndDetect → LLMSayJson → IfElse → SaySync` |
| 无限循环对话 | `Root → Loop(-1) → SayAndDetect → LLMSayJson → IfElse → Loop` |
| 有限重试对话 | `Root → Loop(3) → SayAndDetect → LLMSayJson → IfElse → Loop` |
| 自由对话（非JSON）| `Root → SayAndDetect → LLMSayRaw → SaySync` |
| 播报后结束 | `Root → SaySync(nil)` |

## 完整流程示例（来源: `robots/demo.lua`）

```
Root
 └─ Loop(-1)  ──────────────────── 无限循环
      ├─ SayAndDetect (欢迎+提示, 5s超时)
      │    ├─ fail → Loop  (识别失败，重新循环)
      │    └─ success → LLMSayJson (意图识别)
      │                        ├─ fail → Loop  (LLM失败，重新循环)
      │                        └─ success → IfElse
      │                                      ├─ intent==tell_joke → SaySync("笑话内容")
      │                                      │                        └─ connect → Loop (继续循环)
      │                                      ├─ intent==tell_story → SaySync("故事内容")
      │                                      │                        └─ connect → Loop (继续循环)
      │                                      ├─ intent==bye → nil (结束)
      │                                      └─ else → Loop (未知意图，重新循环)
      └─ fail → nil  (循环次数耗尽，结束)
```

## 节点输出继承链

每个节点在 `do_action()` 中执行 `self.outputs = self.parent_node and self.parent_node.outputs or {}`，然后 `table.insert` 自己的结果。

下游节点通过 `self.outputs[#self.outputs]` 获取最后一个输出：

```lua
-- 获取 SayAndDetect 的 ASR 结果
local asr_result = say_and_detect.output  -- 直接访问节点 output 字段

-- 在 LLMSayJson 的 bind_node_output 中拼接上下文
llm_say:bind_node_output(function(node)
    return say_and_detect.output  -- 传入 ASR 结果给 LLM
end)

-- 在 IfElse 条件函数中访问 LLM 输出
if_else:if_connect(function(self)
    return llm_say.output and llm_say.output.intent == "tell_joke"
end, say_joke)
```

## bind_node_output 绑定模式

`LLMSayJson` 和 `LLMSayRaw` 支持 `bind_node_output`，可将上游节点的 `self.output` 字段动态注入到 LLM 请求上下文中：

```lua
-- 上游节点（如 SayAndDetect）执行后 self.output = ASR识别结果
-- 下游 LLM 节点通过 bind_node_output 访问
llm_say:bind_node_output(function(node)
    -- node 是 LLMSayJson 实例，node.parent_node 是上游节点
    return say_and_detect.output  -- 注入到 LLM 上下文
end)
```

此模式使 LLM 请求体可以动态包含前面所有节点的输出。
