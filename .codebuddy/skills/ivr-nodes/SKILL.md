---
name: ivr-nodes
description: This skill provides IVR (Interactive Voice Response) node components for Lua-based FreeSWITCH call flows. It should be used when the user needs to create, modify, or generate Lua IVR scripts using the node-tree pattern (Root, PlayAndGetDigit, PlayAndGetDigits, PlayAndRequestPost, IfElse, HttpPost, Playback, Loop). Trigger when the user mentions IVR, call flow, voice menu, phone tree, or references any of these node types.
---

# IVR Node Components for Lua

Reusable IVR node component library for building FreeSWITCH Lua call flows using a node-tree architecture.

## Source Files

- **Node definitions**: `ivrs/skill.lua` — reusable library, use `require` to import
- **Usage example**: `ivrs/demo.lua` — complete working IVR script showing all patterns

## How to Use This Skill

When asked to generate IVR Lua code:

1. Understand the call flow requirements (menu options, inputs, API calls, conditions)
2. Choose appropriate nodes from the reference files below
3. Compose them into a complete flow following the patterns from `references/README.md`
4. Output the complete Lua script with: utility functions → config → node definitions → node instances → connections → main loop

## Script Structure (must follow this order)

```lua
-- 1. Utility functions (printTable, deepMerge, event_callback)
-- 2. Config variables (URLs, file paths)
-- 3. Node class definitions (all 8 node types)
-- 4. Node instances (create nodes with meaningful names)
-- 5. Node connections (wire them together)
-- 6. Main loop
```

## Available Nodes

Each node is fully documented in `references/` with definition code + real usage examples from `demo.lua`:

| File | Node | Purpose |
|------|------|---------|
| `references/Root.md` | Root | Flow entry point, defines initial node |
| `references/PlayAndGetDigit.md` | PlayAndGetDigit | Play audio + capture single DTMF digit |
| `references/PlayAndGetDigits.md` | PlayAndGetDigits | Play audio + capture multiple digits |
| `references/PlayAndRequestPost.md` | PlayAndRequestPost | Play audio + HTTP POST request |
| `references/IfElse.md` | IfElse | Conditional branching based on variables |
| `references/HttpPost.md` | HttpPost | HTTP POST request (no audio) |
| `references/Playback.md` | Playback | Play audio file only |
| `references/Loop.md` | Loop | Retry loop with max attempts |
| `references/README.md` | Index | Navigation + quick combo table + full flow diagram |

## Engine API

All nodes have access to these engine methods:

| Method | Description |
|--------|-------------|
| `engine:play_and_get_digit(file, digit, timeout)` | Play audio + capture single digit |
| `engine:play_and_get_digits(file, len, timeout)` | Play audio + capture multi-digit string |
| `engine:play_and_request_post(file, url, body, timeout)` | Play audio + POST, wait response |
| `engine:post_json(url, header, body, timeout)` | Pure JSON POST |
| `engine:playback(file)` | Play audio file |
| `engine:get_uuid()` | Get session UUID |
| `engine:is_ok()` | Check if channel is alive |
| `engine:log(level, msg)` | Log message |
| `engine:sleep(ms)` | Pause execution |

## Key Patterns

### Input → Validate → Loop → Retry
The most common pattern. See `references/Loop.md` for the detailed retry pattern.

### Outputs Chain
Each node inherits `self.outputs` from `self.parent_node.outputs` and appends its result. 
Downstream nodes access results via `self.outputs[#self.outputs]` (last output).
