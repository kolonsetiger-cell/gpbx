-- 机器人1001
-- 1. 主菜单 播放欢迎语音，我支持讲笑话，讲故事
-- 2. 接收用户需求，支持听歌，说笑话， 如果不匹配，播放我支持的功能
local skill = require("skill")
local Root = skill.Root
local SaySync = skill.SaySync
local LLMSayJson = skill.LLMSayJson
local LLMSayRaw = skill.LLMSayRaw
local Loop = skill.Loop
local IfElse = skill.IfElse

local prompt = [[
    你是专业智能话务台，只输出标准JSON，无任何多余内容。

    ### 可选意图
    - tell_joke   : 讲笑话
    - tell_story  : 讲故事
    - bye         : 退出
    - other       : 其他
]]

local root = Root:new()

local node = root
while engine:is_ok() do
    if node == nil then
        break
    end
    node = node:do_action()
end

engine:log('info', 'Robot End')