-- ============================================================
-- 11888 自动化充值 IVR 流程
-- ============================================================

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

-- ============================================================
-- 配置
-- ============================================================
local file_dir = "C:\\Users\\kolonse\\ivrs\\"
local notify_url = "http://127.0.0.1:8083/api/ensure"

-- 提示音文件（需要自行录制，这里使用占位文件名）
local voice_lang      = file_dir .. "11888_select_lang.wav"       -- 选择语言
local voice_card_type = file_dir .. "11888_card_type.wav"         -- 充值付费卡按1
local voice_menu_3    = file_dir .. "11888_menu_3.wav"            -- 按1
local voice_phone_recharge = file_dir .. "11888_phone_recharge.wav" -- 手机充值按1
local voice_menu_5    = file_dir .. "11888_menu_5.wav"            -- 按1
local voice_input_phone = file_dir .. "11888_input_phone.wav"     -- 输入手机号#结束
local voice_confirm   = file_dir .. "11888_confirm.wav"           -- 确认按1，重新输入按2
local voice_input_card = file_dir .. "11888_input_card.wav"       -- 输入18位卡密#结束
local voice_wait      = file_dir .. "11888_wait.wav"              -- 请稍后
local voice_failed    = file_dir .. "11888_failed.wav"            -- 操作不成功

engine:sleep(1000)

-- ============================================================
-- 节点类定义（引用 skill.lua 中的标准节点）
-- ============================================================

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

local PlayAndGetDigit = {}
PlayAndGetDigit.__index = PlayAndGetDigit
function PlayAndGetDigit:new(file, hope_digit, timeout)
    local self = setmetatable({}, PlayAndGetDigit)
    self.file = file
    self.hope_digit = hope_digit
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    self.output = nil
    return self
end

function PlayAndGetDigit:do_action()
    local get_digit = engine:play_and_get_digit(self.file, self.hope_digit, self.timeout)
    self.outputs = self.parent_node.outputs
    if #get_digit == 0 then
        return self.fail_node
    end
    self.output = get_digit
    table.insert(self.outputs, {get_digit})
    return self.success_node
end

function PlayAndGetDigit:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function PlayAndGetDigit:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

local PlayAndGetDigitsWithEnd = {}
PlayAndGetDigitsWithEnd.__index = PlayAndGetDigitsWithEnd
function PlayAndGetDigitsWithEnd:new(file, hope_dtmf, timeout)
    local self = setmetatable({}, PlayAndGetDigitsWithEnd)
    self.file = file
    self.hope_dtmf = hope_dtmf
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    self.output = nil
    return self
end

function PlayAndGetDigitsWithEnd:do_action()
    local get_digits = engine:play_and_get_digits_with_end(self.file, self.hope_dtmf, self.timeout)
    self.outputs = self.parent_node.outputs
    if #get_digits == 0 then
        return self.fail_node
    end
    self.output = get_digits
    table.insert(self.outputs, {result = get_digits})
    return self.success_node
end

function PlayAndGetDigitsWithEnd:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function PlayAndGetDigitsWithEnd:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

local PlayAndRequestPost = {}
PlayAndRequestPost.__index = PlayAndRequestPost
function PlayAndRequestPost:new(file, url, body, timeout)
    local self = setmetatable({}, PlayAndRequestPost)
    self.file = file
    self.url = url
    self.body = body
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    self.bindings = {}
    return self
end

function PlayAndRequestPost:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function PlayAndRequestPost:do_action()
    for _, binding in ipairs(self.bindings) do
        local body = binding(self)
        if body ~= nil then
            deepMerge(self.body, body)
        end
    end

    local response, err = engine:play_and_request_post(self.file, self.url, self.body, self.timeout)
    if err ~= nil then
        return self.fail_node
    end
    self.outputs = self.parent_node.outputs
    table.insert(self.outputs, response)
    return self.success_node
end

function PlayAndRequestPost:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function PlayAndRequestPost:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

local IfElse = {}
IfElse.__index = IfElse
function IfElse:new()
    local self = setmetatable({}, IfElse)
    self.condition = nil
    self.true_node = nil
    self.elseif_node = {}
    self.else_node = nil
    self.inputs = nil
    return self
end

function IfElse:do_action()
    self.outputs = self.parent_node.outputs
    if self.condition(self) then
        return self.true_node
    end
    for i, v in ipairs(self.elseif_node) do
        if v.condition(self) then
            return v.node
        end
    end
    return self.else_node
end

function IfElse:if_connect(condition, node)
    self.true_node = node
    self.condition = condition
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function IfElse:else_connect(node)
    self.else_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function IfElse:ifelse_connect(condition, node)
    self.else_node = node
    node.parent_node = self
    table.insert(self.elseif_node, {condition = condition, node = node})
    return self
end

local HttpPost = {}
HttpPost.__index = HttpPost
function HttpPost:new(url, header, body, timeout)
    local self = setmetatable({}, PlayAndGetDigit)
    self.url = url
    self.header = header
    self.body = body
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    self.bindings = {}
    return self
end

function HttpPost:bind_node_output(func_bind)
    table.insert(self.bindings, func_bind)
end

function HttpPost:do_action()
    for _, binding in ipairs(self.bindings) do
        local body = binding(self)
        if body ~= nil then
            deepMerge(self.body, body)
        end
    end

    local code, response, err = engine:post_json(self.url, {}, self.body, 10000)
    self.outputs = self.parent_node.outputs
    if code ~= 200 or err ~= nil then
        return self.fail_node
    end
    table.insert(self.outputs, {response})
    return self.success_node
end

function HttpPost:success_connect(node)
    self.success_node = node
    if node ~= nil then
        return self
    end
    node.parent_node = self
    return self
end

function HttpPost:fail_connect(node)
    self.fail_node = node
    if node ~= nil then
        return self
    end
    node.parent_node = self
    return self
end

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
    if node ~= nil then
        return self
    end
    node.parent_node = self
    return self
end

local Loop = {}
Loop.__index = Loop
function Loop:new(loop_count)
    local self = setmetatable({}, Playback)
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
    if node ~= nil then
        return self
    end
    node.parent_node = self.parent_node
    return self
end

function Loop:fail_connect(node)
    self.fail_node = node
    return self
end

-- ============================================================
-- 节点实例
-- ============================================================

-- Step 1: 选择语言，按1选普通话
local step1_lang = PlayAndGetDigit:new(voice_lang, "1", 10000)

-- Step 2: 充值付费卡按1
local step2_card = PlayAndGetDigit:new(voice_card_type, "1", 10000)

-- Step 3: 按1
local step3_menu = PlayAndGetDigit:new(voice_menu_3, "1", 10000)

-- Step 4: 手机充值按1
local step4_recharge = PlayAndGetDigit:new(voice_phone_recharge, "1", 10000)

-- Step 5: 按1
local step5_menu = PlayAndGetDigit:new(voice_menu_5, "1", 10000)

-- Step 6: 为本机充值按1#，其他手机输入号码#结束
local step6_phone = PlayAndGetDigitsWithEnd:new(voice_input_phone, "#", 30000)
local step6_phone_ifelse = IfElse:new()

-- Step 7: 输入手机号码#结束（step6 输入非1时走这里）
local step7_phone_confirm = PlayAndGetDigitsWithEnd:new(voice_input_phone, "#", 30000)

-- Step 8: 确认按1，重新输入按2
local step8_confirm = PlayAndGetDigit:new(voice_confirm, "12", 10000)
local step8_confirm_ifelse = IfElse:new()

-- Step 9: 按1（确认后）
local step9_confirm = PlayAndGetDigit:new(voice_menu_5, "1", 10000)

-- Step 10: 输入18位卡密#结束
local step10_card_pwd = PlayAndGetDigitsWithEnd:new(voice_input_card, "#", 30000)

-- Step 11: 请稍后 → 操作不成功
local step11_wait = Playback:new(voice_wait)
local step11_failed = Playback:new(voice_failed)

-- 重新输入手机号的 Loop（step8 按2 时回到 step6）
local phone_retry_loop = Loop:new(3)

-- 通知节点（将充值信息上报后端）
local node_notify = HttpPost:new(notify_url, {}, {session_id = engine:get_uuid()}, 10000)

-- ============================================================
-- 节点连接
-- ============================================================

local root = Root:new()

-- Step 1 → Step 2
root:connect(step1_lang)
step1_lang:success_connect(step2_card)
step1_lang:fail_connect(nil)

-- Step 2 → Step 3
step2_card:success_connect(step3_menu)
step2_card:fail_connect(nil)

-- Step 3 → Step 4
step3_menu:success_connect(step4_recharge)
step3_menu:fail_connect(nil)

-- Step 4 → Step 5
step4_recharge:success_connect(step5_menu)
step4_recharge:fail_connect(nil)

-- Step 5 → Step 6
step5_menu:success_connect(step6_phone)
step5_menu:fail_connect(nil)

-- Step 6: 输入手机号（1#为本机，其他为别的号码）
-- 判断：如果输入是 "1" 则为本机充值，直接到 Step 9
--        如果输入不是 "1" 则是其他手机号，到 Step 7 再输一遍确认
step6_phone:success_connect(step6_phone_ifelse)
step6_phone:fail_connect(nil)

-- Step 6 分支判断：输入"1"（本机）→ Step 9，否则 → Step 7
step6_phone_ifelse:if_connect(function(self)
    return step6_phone.output == "1"
end, step9_confirm)  -- 本机充值，直接到确认
step6_phone_ifelse:else_connect(step7_phone_confirm)  -- 其他手机号 → 再输一遍

-- Step 7 → Step 8 确认
step7_phone_confirm:success_connect(step8_confirm)
step7_phone_confirm:fail_connect(nil)

-- Step 8: 确认按1，重新输入按2
step8_confirm:success_connect(step8_confirm_ifelse)
step8_confirm:fail_connect(nil)

-- Step 8 确认分支：按1 → Step 9，按2 → Loop 重新输入
step8_confirm_ifelse:if_connect(function(self)
    return step8_confirm.output == "1"
end, step9_confirm)
step8_confirm_ifelse:ifelse_connect(function(self)
    return step8_confirm.output == "2"
end, phone_retry_loop)
step8_confirm_ifelse:else_connect(nil)

-- 重新输入 Loop
phone_retry_loop:connect(step6_phone)
phone_retry_loop:fail_connect(nil)

-- Step 9 → Step 10
step9_confirm:success_connect(step10_card_pwd)
step9_confirm:fail_connect(nil)

-- Step 10 → 通知后端 → Step 11
step10_card_pwd:success_connect(node_notify)
step10_card_pwd:fail_connect(nil)

-- 通知（上报充值信息）
node_notify:bind_node_output(function(self)
    local phone = step6_phone.output
    if phone ~= "1" and step7_phone_confirm.output then
        phone = step7_phone_confirm.output
    elseif phone == "1" then
        phone = "本机"
    end
    return {
        phone = phone,
        card_password = step10_card_pwd.output,
        action = "recharge_11888"
    }
end)
node_notify:success_connect(step11_wait)
node_notify:fail_connect(step11_wait)  -- 通知失败也继续

-- Step 11: 请稍后 → 操作不成功 → 结束
step11_wait:connect(step11_failed)
step11_failed:connect(nil)

-- ============================================================
-- 主循环
-- ============================================================

local node = root
while engine:is_ok() do
    if node == nil then
        break
    end
    node = node:do_action()
end

engine:log('info', '11888 IVR End')
