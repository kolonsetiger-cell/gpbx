-- robots/skill.lua
-- 机器人 Skill 节点库
-- 用法：local skill = require("robots/skill")  -- 根据实际 script_dir 配置调整路径
-- engine API（由 FreeSWITCH Lua 运行环境注入）：
--   engine:say_and_detect(text, timeout) -> (response, err)
--   engine:say_sync(text)                   -> (response, err)
--   engine:llm_say_json(prompt, text, timeout) -> (response, err)
--   engine:llm_say_raw(prompt, text, timeout)  -> (response, err)
--   engine:is_ok()  -> boolean
--   engine:log(level, msg)
--   engine:hangup()

local skill = {}

-- ==================== Root ====================
-- 根节点，连接到第一个要执行的节点
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

-- 连接下一个节点（成功分支）
function Root:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return Root
end

skill.Root = Root

-- ==================== SayAndDetect ====================
-- 播放 TTS 提示语音，等待用户说话并 ASR 识别
-- 成功 -> success_node，失败/超时 -> fail_node
local SayAndDetect = {}
SayAndDetect.__index = SayAndDetect

function SayAndDetect:new(text, timeout)
    local self = setmetatable({}, SayAndDetect)
    self.text     = text     -- TTS 提示文本
    self.timeout  = timeout  -- ASR 等待超时（毫秒）
    self.parent_node = nil
    self.outputs     = nil
    self.output      = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    return self
end

function SayAndDetect:do_action()
    local response, err = engine:say_and_detect(self.text, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function SayAndDetect:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function SayAndDetect:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.SayAndDetect = SayAndDetect

-- ==================== SaySync ====================
-- 同步 TTS 播报（等待播报完成后返回 ASR 结果）
-- 成功 -> success_node（通过 connect 设置），失败 -> fail_node
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

-- SaySync 使用 connect 设置成功节点（与 SayAndDetect 接口保持一致）
function SaySync:connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.SaySync = SaySync

-- ==================== LLMSayJson ====================
-- 将前面节点的输出通过 bindings 拼接为 context，
-- 调用 LLM 并解析返回的 JSON 结果
-- 成功 -> success_node，失败 -> fail_node
local LLMSayJson = {}
LLMSayJson.__index = LLMSayJson

function LLMSayJson:new(prompt, timeout)
    local self = setmetatable({}, LLMSayJson)
    self.prompt   = prompt
    self.timeout  = timeout
    self.parent_node  = nil
    self.outputs      = nil
    self.output       = nil
    self.error        = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings     = {}  -- bind_node_output 注册的回调函数列表
    return self
end

-- 注册绑定函数：将其他节点的 output 拼接到 LLM 请求上下文中
function LLMSayJson:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function LLMSayJson:do_action()
    local text = ""
    for _, binding in ipairs(self.bindings) do
        local body = binding(self)
        if body ~= nil then
            text = text .. body
        end
    end
    local response, err = engine:llm_say_json(self.prompt, text, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function LLMSayJson:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function LLMSayJson:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.LLMSayJson = LLMSayJson

-- ==================== LLMSayRaw ====================
-- 与 LLMSayJson 接口一致，但 LLM 返回原始文本（非 JSON）
local LLMSayRaw = {}
LLMSayRaw.__index = LLMSayRaw

function LLMSayRaw:new(prompt, timeout)
    local self = setmetatable({}, LLMSayRaw)
    self.prompt   = prompt
    self.timeout  = timeout
    self.parent_node  = nil
    self.outputs      = nil
    self.output       = nil
    self.error        = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings     = {}
    return self
end

function LLMSayRaw:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function LLMSayRaw:do_action()
    local text = ""
    for _, binding in ipairs(self.bindings) do
        local body = binding(self)
        if body ~= nil then
            text = text .. body
        end
    end
    local response, err = engine:llm_say_raw(self.prompt, text, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, { response })
    return self.success_node
end

function LLMSayRaw:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function LLMSayRaw:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.LLMSayRaw = LLMSayRaw

-- ==================== Loop ====================
-- 循环节点：loop_count = -1 表示无限循环，
-- 0 表示直接跳到 fail_node，>0 表示剩余循环次数
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

-- 连接循环体节点
function Loop:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

-- 连接循环退出节点（loop_count 耗尽时跳转）
function Loop:fail_connect(node)
    self.fail_node = node
    return self
end

skill.Loop = Loop

-- ==================== IfElse ====================
-- 条件分支节点
local IfElse = {}
IfElse.__index = IfElse

function IfElse:new()
    local self = setmetatable({}, IfElse)
    self.condition    = nil
    self.true_node    = nil
    self.elseif_node  = {}  -- 格式：{ condition = func, node = node }
    self.else_node    = nil
    self.inputs       = nil
    return self
end

function IfElse:do_action()
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if self.condition and self.condition(self) then
        return self.true_node
    end
    for _, v in ipairs(self.elseif_node) do
        if v.condition and v.condition(self) then
            return v.node
        end
    end
    return self.else_node
end

-- 设置 if 分支
function IfElse:if_connect(condition, node)
    self.true_node = node
    self.condition = condition
    if node == nil then return self end
    node.parent_node = self
    return self
end

-- 设置 else 分支
function IfElse:else_connect(node)
    self.else_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

-- 添加 elseif 分支（可多次调用）
function IfElse:ifelse_connect(condition, node)
    node.parent_node = self
    table.insert(self.elseif_node, { condition = condition, node = node })
    return self
end

skill.IfElse = IfElse

return skill
