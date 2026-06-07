-- 机器人 demo
-- 功能：播放欢迎语音，接收用户意图（讲笑话/讲故事/退出），路由到对应节点
-- 用法：在 FreeSWITCH 中通过 robot 配置加载此脚本
local skill = require("robots/skill")  -- 根据实际 script_dir 调整路径
local Root         = skill.Root
local SayAndDetect = skill.SayAndDetect
local LLMSayJson   = skill.LLMSayJson   -- 注意：skill.lua 导出键为 LLMSayJson（两个 L）
local LLMSayRaw    = skill.LLMSayRaw
local Loop         = skill.Loop
local IfElse       = skill.IfElse
local SaySync      = skill.SaySync

-- LLM prompt：用于意图识别
local prompt = [[
你是专业智能话务台，只输出标准JSON，无任何多余内容。

### 可选意图
- tell_joke   : 讲笑话
- tell_story  : 讲故事
- bye         : 退出
- other       : 其他
]]

-- 构建节点树
local root = Root:new()
local loop = Loop:new(-1)  -- 无限循环，等待用户输入
local say_and_detect = SayAndDetect:new(
    "欢迎使用我的机器人服务，我支持讲笑话，讲故事，告诉我您想做什么", 5000)
local llm_say = LLMSayJson:new(prompt, 5000)
local if_else = IfElse:new()
local say_joke  = SaySync:new("大明捡了2分钱，但是钱存了")
local say_story = SaySync:new("小明捡了一分钱，但是钱没了。")

-- 连接节点树
root:connect(loop)
loop:connect(say_and_detect)
loop:fail_connect(nil)  -- 循环体执行失败则退出循环

say_and_detect:success_connect(llm_say)
say_and_detect:fail_connect(loop)  -- ASR 失败，重新循环

-- 将 SayAndDetect 的输出绑定到 LLM 上下文
llm_say:bind_node_output(function(node)
    return say_and_detect.output
end)

llm_say:success_connect(if_else)
llm_say:fail_connect(loop)

-- 意图路由
if_else:if_connect(function(node)
    return llm_say.output and llm_say.output.intent == "tell_joke"
end, say_joke)

if_else:ifelse_connect(function(node)
    return llm_say.output and llm_say.output.intent == "tell_story"
end, say_story)

if_else:ifelse_connect(function(node)
    return llm_say.output and llm_say.output.intent == "bye"
end, nil)  -- bye：结束（返回 nil 退出主循环）

if_else:else_connect(loop)  -- 其他意图：重新循环

-- 执行节点树
local node = root
while engine:is_ok() do
    if node == nil then
        break
    end
    node = node:do_action()
end

engine:log('info', 'Robot End')
