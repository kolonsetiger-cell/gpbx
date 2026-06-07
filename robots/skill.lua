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


local SayAndDetect = {}
SayAndDetect.__index = SayAndDetect
function SayAndDetect:new(text, timeout)
    local self = setmetatable({}, SayAndDetect)
    self.text = text
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.output = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function SayAndDetect:do_action()
    local response, err = engine:say_and_detect(self.text, self.timeout)
    self.outputs = self.parent_node.outputs
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, {response})
    return self.success_node
end

function SayAndDetect:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function SayAndDetect:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

local SaySync = {}
SaySync.__index = SaySync
function SaySync:new(text)
    local self = setmetatable({}, SaySync)
    self.text = text
    self.parent_node = nil
    self.outputs = nil
    self.output = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function SaySync:do_action()
    local response, err = engine:say_sync(self.text)
    self.outputs = self.parent_node.outputs
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, {response})
    return self.success_node
end

function SaySync:connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end


local LLMSayJson = {}
LLMSayJson.__index = LLMSayJson
function LLMSayJson:new(prompt, text, timeout)
    local self = setmetatable({}, LLMSayJson)
    self.prompt = prompt
    self.text = text
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.output = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function LLMSayJson:do_action()
    local response, err = engine:llm_say_json(self.prompt, self.text, self.timeout)
    self.outputs = self.parent_node.outputs
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, {response})
    return self.success_node
end

function LLMSayJson:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function LLMSayJson:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

local LLMSayRaw = {}
LLMSayRaw.__index = LLMSayRaw
function LLMSayRaw:new(prompt, text, timeout)
    local self = setmetatable({}, LLMSayRaw)
    self.prompt = prompt
    self.text = text
    self.timeout = timeout
    self.parent_node = nil
    self.outputs = nil
    self.output = nil
    self.error = nil
    self.success_node = nil
    self.fail_node = nil
    return self
end

function LLMSayRaw:do_action()
    local response, err = engine:llm_say_raw(self.prompt, self.text, self.timeout)
    self.outputs = self.parent_node.outputs
    if err ~= nil or response == nil or #response == 0 then
        return self.fail_node
    end
    self.output = response
    table.insert(self.outputs, {response})
    return self.success_node
end

function LLMSayRaw:success_connect(node)
    self.success_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

function LLMSayRaw:fail_connect(node)
    self.fail_node = node
    if node == nil then
        return self
    end
    node.parent_node = self
    return self
end

local Loop = {}
Loop.__index = Loop
function Loop:new(loop_count)
    local self = setmetatable({}, Loop)
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

return {
    Root = Root,
    SaySync = SaySync,
    LLMSayJson = LLMSayJson,
    LLMSayRaw = LLMSayRaw,
    Loop = Loop,
    IfElse = IfElse,
}