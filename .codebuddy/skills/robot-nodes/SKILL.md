---
name: robot-nodes
description: This skill provides robot (AI assistant) node components for Lua-based FreeSWITCH call flows. It should be used when the user needs to create, modify, or generate Lua robot scripts using the node-tree pattern (Root, SayAndDetect, SaySync, LLMSayJson, LLMSayRaw, IfElse, Loop). Trigger when the user mentions robot, AI assistant, voice bot, LLM call flow, or references any of these node types.
---

# Robot Node Components for Lua

Reusable robot node component library for building FreeSWITCH Lua AI assistant call flows using a node-tree architecture.

## Source Files

- **Node definitions**: `robots/skill.lua` — reusable library, use `require("robots/skill")` to import
- **Usage example**: `robots/demo.lua` — complete working robot script showing all patterns

## How to Use This Skill

When asked to generate robot Lua code:

1. Understand the call flow requirements (welcome message, intent recognition, responses)
2. Choose appropriate nodes from the reference files below
3. Compose them into a complete flow following the patterns from `references/README.md`
4. Output the complete Lua script with: require → config → node instances → connections → main loop

## Script Structure (must follow this order)

```lua
-- 1. Require skill library
local skill = require("robots/skill")

-- 2. Config variables (prompts, timeouts)
local prompt = [[...]]

-- 3. Node instances (create nodes with meaningful names)
local root = skill.Root:new()
local say_and_detect = skill.SayAndDetect:new("欢迎...", 5000)
-- ...

-- 4. Node connections (wire them together)
root:connect(loop)
say_and_detect:success_connect(llm_say)
-- ...

-- 5. Main loop
local node = root
while engine:is_ok() do
    if node == nil then break end
    node = node:do_action()
end
```

## Available Nodes

Each node is fully documented in `references/` with definition code + real usage examples from `demo.lua`:

| File | Node | Purpose |
|------|------|---------|
| `references/Root.md` | Root | Flow entry point, defines initial node |
| `references/SayAndDetect.md` | SayAndDetect | Play TTS + ASR recognition |
| `references/SaySync.md` | SaySync | Play TTS synchronously + ASR |
| `references/LLMSayJson.md` | LLMSayJson | Call LLM, parse JSON response |
| `references/LLMSayRaw.md` | LLMSayRaw | Call LLM, return raw text |
| `references/IfElse.md` | IfElse | Conditional branching based on variables |
| `references/Loop.md` | Loop | Retry loop with max attempts |

## Engine API

All nodes have access to these engine methods:

| Method | Description |
|--------|-------------|
| `engine:say_and_detect(text, timeout)` | Play TTS + wait for ASR input |
| `engine:say_sync(text)` | Play TTS synchronously + return ASR input |
| `engine:llm_say_json(prompt, text, timeout)` | Call LLM, parse JSON response |
| `engine:llm_say_raw(prompt, text, timeout)` | Call LLM, return raw text |
| `engine:is_ok()` | Check if channel is alive |
| `engine:log(level, msg)` | Log message |
| `engine:hangup()` | Hang up the call |

## Key Patterns

### Welcome → ASR → LLM → Intent Routing
The most common robot pattern. See `references/IfElse.md` for intent routing examples.

### Outputs Chain
Each node inherits `self.outputs` from `self.parent_node.outputs` and appends its result.
Downstream nodes access results via `node.output` (direct access) or `self.outputs[#self.outputs]`.

### Infinite Loop for Continuous Interaction
Use `Loop:new(-1)` for the main interaction loop, and route to `nil` when user says "bye".
