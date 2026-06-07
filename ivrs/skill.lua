-- ivrs/skill.lua
-- IVR Skill 节点库
-- 用法：local skill = require("ivrs/skill")  -- 根据实际 script_dir 配置调整路径
-- engine API（由 FreeSWITCH Lua 运行环境注入）：
--   engine:play_and_get_digit(file, hope_digit, timeout)   -> digit
--   engine:play_and_get_digits(file, hope_len, timeout)      -> digits
--   engine:play_and_get_digits_with_end(file, hope_dtmf, timeout) -> digits
--   engine:play_and_request_post(file, url, body, timeout) -> (response, err)
--   engine:post_json(url, header, body, timeout)           -> (code, response, err)
--   engine:playback(file)
--   engine:sleep(ms)
--   engine:set_callback(fn)
--   engine:get_uuid() -> string
--   engine:log(level, msg)
--   engine:is_ok() -> boolean

local skill = {}

-- ==================== 工具函数 ====================
-- 深合并两个 table（t2 覆盖 t1 的同名字段，子 table 递归合并）
local function deepMerge(t1, t2)
    local result = {}
    for k, v in pairs(t1) do
        result[k] = v
    end
    for k, v in pairs(t2) do
        if type(v) == "table" and type(result[k]) == "table" then
            result[k] = deepMerge(result[k], v)
        else
            result[k] = v
        end
    end
    return result
end

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

function Root:connect(node)
    self.next_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return Root
end

skill.Root = Root

-- ==================== PlayAndGetDigit ====================
-- 播放语音文件，等待用户按 1 位 DTMF 键
-- hope_digit：期望的按键（如 "1"），空字符串表示接受任意 1 位
local PlayAndGetDigit = {}
PlayAndGetDigit.__index = PlayAndGetDigit

function PlayAndGetDigit:new(file, hope_digit, timeout)
    local self = setmetatable({}, PlayAndGetDigit)
    self.file        = file
    self.hope_digit  = hope_digit
    self.timeout     = timeout
    self.parent_node = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    return self
end

function PlayAndGetDigit:do_action()
    local get_digit = engine:play_and_get_digit(self.file, self.hope_digit, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if #get_digit == 0 then
        return self.fail_node
    end
    table.insert(self.outputs, { get_digit })
    return self.success_node
end

function PlayAndGetDigit:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function PlayAndGetDigit:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.PlayAndGetDigit = PlayAndGetDigit

-- ==================== PlayAndGetDigits ====================
-- 播放语音文件，等待用户按指定长度的 DTMF（无结束符）
local PlayAndGetDigits = {}
PlayAndGetDigits.__index = PlayAndGetDigits

function PlayAndGetDigits:new(file, hope_len, timeout)
    local self = setmetatable({}, PlayAndGetDigits)
    self.file      = file
    self.hope_len  = hope_len
    self.timeout   = timeout
    self.parent_node  = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.output      = nil
    return self
end

function PlayAndGetDigits:do_action()
    local get_digits = engine:play_and_get_digits(self.file, self.hope_len, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if #get_digits == 0 then
        return self.fail_node
    end
    self.output = get_digits
    table.insert(self.outputs, { result = get_digits })
    return self.success_node
end

function PlayAndGetDigits:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function PlayAndGetDigits:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.PlayAndGetDigits = PlayAndGetDigits

-- ==================== PlayAndGetDigitsWithEnd ====================
-- 播放语音文件，等待用户按 DTMF，以指定结束符（如 #）结尾
local PlayAndGetDigitsWithEnd = {}
PlayAndGetDigitsWithEnd.__index = PlayAndGetDigitsWithEnd

function PlayAndGetDigitsWithEnd:new(file, hope_dtmf, timeout)
    local self = setmetatable({}, PlayAndGetDigitsWithEnd)
    self.file       = file
    self.hope_dtmf  = hope_dtmf  -- 结束符，如 "#"
    self.timeout    = timeout
    self.parent_node  = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.output      = nil
    return self
end

function PlayAndGetDigitsWithEnd:do_action()
    local get_digits = engine:play_and_get_digits_with_end(self.file, self.hope_dtmf, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if #get_digits == 0 then
        return self.fail_node
    end
    self.output = get_digits
    table.insert(self.outputs, { result = get_digits })
    return self.success_node
end

function PlayAndGetDigitsWithEnd:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function PlayAndGetDigitsWithEnd:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.PlayAndGetDigitsWithEnd = PlayAndGetDigitsWithEnd

-- ==================== PlayAndRequestPost ====================
-- 播放语音文件（等待按键），同时向 URL 发起 POST 请求，
-- 将响应作为节点输出
-- 可通过 bind_node_output 绑定前面节点的输出，合并到 POST body 中
local PlayAndRequestPost = {}
PlayAndRequestPost.__index = PlayAndRequestPost

function PlayAndRequestPost:new(file, url, body, timeout)
    local self = setmetatable({}, PlayAndRequestPost)
    self.file     = file
    self.url      = url
    self.body     = body or {}
    self.timeout  = timeout
    self.parent_node = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings    = {}  -- bind_node_output 注册的回调函数列表
    return self
end

function PlayAndRequestPost:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function PlayAndRequestPost:do_action()
    -- 合并前面节点的输出到 body
    for _, binding in ipairs(self.bindings) do
        local body_part = binding(self)
        if body_part ~= nil then
            self.body = deepMerge(self.body, body_part)
        end
    end

    local response, err = engine:play_and_request_post(self.file, self.url, self.body, self.timeout)
    if err ~= nil then
        return self.fail_node
    end
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    table.insert(self.outputs, response)
    return self.success_node
end

function PlayAndRequestPost:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function PlayAndRequestPost:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.PlayAndRequestPost = PlayAndRequestPost

-- ==================== IfElse ====================
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

function IfElse:if_connect(condition, node)
    self.true_node = node
    self.condition = condition
    if node == nil then return self end
    node.parent_node = self
    return self
end

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

-- ==================== HttpPost ====================
-- 直接向 URL 发起 JSON POST 请求（不播放语音）
-- 可通过 bind_node_output 绑定前面节点的输出，合并到 POST body 中
local HttpPost = {}
HttpPost.__index = HttpPost

function HttpPost:new(url, header, body, timeout)
    local self = setmetatable({}, HttpPost)
    self.url      = url
    self.header   = header or {}
    self.body     = body or {}
    self.timeout  = timeout or 10000
    self.parent_node = nil
    self.outputs     = nil
    self.error       = nil
    self.success_node = nil
    self.fail_node    = nil
    self.bindings     = {}  -- 修复原 bug：原代码缺少 bindings 初始化
    return self
end

function HttpPost:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function HttpPost:do_action()
    -- 合并前面节点的输出到 body
    for _, binding in ipairs(self.bindings) do
        local body_part = binding(self)
        if body_part ~= nil then
            self.body = deepMerge(self.body, body_part)
        end
    end

    local code, response, err = engine:post_json(self.url, self.header, self.body, self.timeout)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    if code ~= 200 or err ~= nil then
        return self.fail_node
    end
    table.insert(self.outputs, { response })
    return self.success_node
end

function HttpPost:success_connect(node)
    self.success_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

function HttpPost:fail_connect(node)
    self.fail_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.HttpPost = HttpPost

-- ==================== Playback ====================
-- 播放语音文件（不等待按键，不等待 DTMF）
local Playback = {}
Playback.__index = Playback

function Playback:new(file)
    local self = setmetatable({}, Playback)
    self.file = file
    self.parent_node = nil
    self.outputs    = nil
    self.error      = nil
    self.next_node  = nil
    return self
end

function Playback:do_action()
    engine:playback(self.file)
    self.outputs = self.parent_node and self.parent_node.outputs or {}
    return self.next_node
end

function Playback:connect(node)
    self.next_node = node
    if node == nil then return self end
    node.parent_node = self
    return self
end

skill.Playback = Playback

-- ==================== Loop ====================
-- 循环节点：loop_count = -1 无限循环，
-- 0 跳到 fail_node，>0 每次减 1
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
    if node == nil then return self end
    node.parent_node = self
    return self
end

function Loop:fail_connect(node)
    self.fail_node = node
    return self
end

skill.Loop = Loop

return skill
