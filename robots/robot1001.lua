-- 机器人1001
-- 1. 主菜单 播放欢迎语音，我支持讲笑话，讲故事
-- 2. 接收用户需求，支持听歌，说笑话， 如果不匹配，播放我支持的功能

local prompt = [[
    你是专业智能话务台，只输出标准JSON，无任何多余内容。

    ### 可选意图
    - tell_joke   : 讲笑话
    - tell_story  : 讲故事
    - bye         : 退出
    - other       : 其他
]]

function printTable(t, indent)
    indent = indent or 0
    local prefix = string.rep("  ", indent)  -- 缩进
    local res = ""
    for k, v in pairs(t) do
        if type(v) == "table" then
            res = res .. prefix .. tostring(k) .. " = {\n"
            res = res .. printTable(v, indent + 1)
            res = res .. prefix .. "}\n"
        else
            res = res .. prefix .. tostring(k) .. " = " .. tostring(v) .. "\n"
        end
    end
    return res
end

local menu = {
    {
        name = "start",
        say_and_check = {
            say = "欢迎使用我的机器人服务，我支持讲笑话，讲故事，告诉我您想做什么",
            hope = {
                {
                    "tell_joke", "say_joke"
                },
                {
                    "tell_story", "say_story"
                },
                {
                    "bye", "exit"
                }
            },
            not_found = {
                say = "抱歉我没有听清，请再说一次",
                retry_times = 3
            },
        },
        error = "exit"
    },
    {
        name = "say_joke",
        say = {
            say = "发动机付款了房间看电视啦附近开了房间的发射点立刻解放拉萨尽快答复螺丝钉解放了发大水家乐福就开始多了几分但是",
            say_end = "start",
        },
        error = "exit"
    },
    {
        name = "say_story",
        say = {
            say = "呵呵呵呵呵呵呵呵佛挡杀佛还打算付款记录飞机按时灯笼裤飞机的萨拉飞机开绿灯是否发的撒娇了付款就两点十六分随风倒士大夫角度来思考附件",
            say_end = "start",
        },
        error = "exit"
    },
    {
        name = "exit",
        say = {
            say = "谢谢使用，再见",
        },
    }
}

local function parse_menu(menu_detail)
    local menu_table = {}
    for _, v in ipairs(menu_detail) do
        menu_table[v.name] = v
    end
    return menu_table
end

local function run_action(menu_t, action)
    if action.say then
        engine:say_sync(action.say.say)
        return menu_t[action.say.say_end]
    end

    if action.say_and_check then
        local retry_times = 0;
        local max_retry = 1;
        if action.say_and_check.not_found and action.say_and_check.not_found.retry_times then
            max_retry = action.say_and_check.not_found.retry_times
        end
        while retry_times < max_retry
        do
            local asr, err = engine:say_and_detect(action.say_and_check.say, 5000)
            if asr ~= nil then
                engine:log('info', 'asr = ' .. asr)
            end
            if err ~= nil then
                engine:log('info', 'err = ' .. err)
            end
            if err ~= nil or #asr == 0 then
                engine:say_sync(action.say_and_check.not_found.say)
                -- return menu_t[action.say_and_check.not_found.retry_times]
                retry_times = retry_times + 1
                engine:log('info', 'retry_times = ' .. retry_times)
            else
                -- do llm_say_json to get intent
                -- if intent is not found, then return error_message
                local res, err = engine:llm_say_json(prompt, asr, 5000)
                if err ~= nil then
                    return menu_t[action.error]
                end
                engine:log('info', printTable(res, 1))
                -- engine:log('info', 'intent = ' .. res)
                for _, v in ipairs(action.say_and_check.hope) do
                    if v[1] == res.intent then
                        return menu_t[v[2]]
                    end
                end
                engine:say_sync(action.say_and_check.not_found.say)
                retry_times = retry_times + 1
                engine:log('info', 'retry_times = ' .. retry_times)
            end
        end
        return menu_t[action.error]
    end

    return menu_t[action.error]
end

local memu_table = parse_menu(menu)
local action = memu_table["start"]
while engine:is_ok() do
    action = run_action(memu_table, action)
    if action == nil then
        engine:log('info', 'ivr end')
        break
    end
end

engine:log('info', 'Robot End')
