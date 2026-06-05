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
    return self
end

function PlayAndGetDigit:do_action()
    local get_digit = engine:play_and_get_digit(self.file, self.hope_digit, self.timeout)
    self.outputs = self.parent_node.outputs
    if #get_digit == 0 then
        return self.fail_node
    end
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

local PlayAndGetDigits = {}
PlayAndGetDigits.__index = PlayAndGetDigits
function PlayAndGetDigits:new(file, hope_len, timeout)
    local self = setmetatable({}, PlayAndGetDigits)
    self.file = file
    self.hope_len = hope_len
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function PlayAndGetDigits:do_action()
    local get_digits = engine:play_and_get_digits(self.file, self.hope_len, self.timeout)
    self.outputs = self.parent_node.outputs
    if #get_digits == 0 then
        return self.fail_node
    end
    table.insert(self.outputs, {result = get_digits})
    return self.success_node
end

function PlayAndGetDigits:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function PlayAndGetDigits:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end
-- play_and_request_post
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
    return self
end

-- function PlayAndRequestPost:use_last_node_output(keys)
-- end

function PlayAndRequestPost:do_action()
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
    return self
end

function HttpPost:do_action()
    local code, response, err = engine:post_json(self.url, {}, {session_id = engine:get_uuid()}, 10000)
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
