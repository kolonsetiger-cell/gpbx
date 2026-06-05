-- 机器人1001
-- 1. 主菜单 播放欢迎语音，我支持讲笑话，讲故事
-- 2. 接收用户需求，支持听歌，说笑话， 如果不匹配，播放我支持的功能

local welcome = "欢迎使用我的机器人服务，我支持讲笑话，讲故事，告诉我您想做什么"

local prompt = [[
    你是专业智能话务台，只输出标准JSON，无任何多余内容。

    ### 可选意图
    - tell_joke   : 讲笑话
    - tell_story  : 讲故事
    - other       : 其他
]]

local big_mode_url = "http://10.55.29.142:1234/api/v1/chat"
local big_model = {
    model= "qwen2.5-vl-7b-instruct",
    input= "",
    context_length = 8000,
    temperature = 0,
    system_prompt = prompt
}

local response_id = ""
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

local say_joke_menu = {
    do_action = function()
        engine:say_sync("呵呵呵呵呵呵呵呵佛挡杀佛还打算付款记录飞机按时灯笼裤飞机的萨拉飞机开绿灯是否发的撒娇了付款就两点十六分随风倒士大夫角度来思考附件")
        engine:say_sync("这个笑话是不是很好听呢，很高兴让您开心，谢谢使用")
        return first_menu
    end
}

local say_story_menu = {
    do_action = function()
        engine:say_sync("警方打开拉萨解放开绿灯撒解放开绿灯撒加发动机是卡洛夫可见度撒了飞机拉萨大家发发的艰苦拉萨附近开了的洒家分厘卡圣诞节符合你打算克里夫就会打开拉萨飓风艾娃")
        engine:say_sync("这个故事是不是很好听呢，很高兴让您开心，谢谢使用")
        return first_menu
    end
}

local server_internal_err_menu = {
    do_action = function()
        engine:say_sync("服务器内部错误，请稍后重试，谢谢使用")
        engine:hangup()
    end
}

local hear_error_multi_times_menu = {
    do_action = function()
        engine:say_sync("我无法理解您的需求，谢谢使用。")
        engine:hangup()
    end
}

local not_find_intent_menu = {
    times = 0,
    do_action = function()
        local asr, err = engine:say_and_detect("请告诉我您想要做什么？我支持讲笑话，讲故事。您可以说讲个笑话，讲个故事。", 0)
        if err ~= nil then
            engine:log('error', 'say_and_detect error: ' .. err)
            return server_internal_err_menu
        else
            engine:log('info', 'say_and_detect result: ' .. asr)
        end
        if string.len(asr) == 0 then
            engine:log('error', 'say_and_detect result is empty')
            not_find_intent_menu.times = not_find_intent_menu.times + 1
            if not_find_intent_menu.times > 3 then
                return hear_error_multi_times_menu
            end
            return not_find_intent_menu
        end
        big_model.input = asr
        big_model.response_id = response_id
        local code, res, err = engine:post_json(big_mode_url, {}, big_model, 5000)
        if err ~= nil then
            engine:log('error', 'post_json error: ' .. err)
            return server_internal_err_menu
        else
            if code ~= 200 then
                engine:log('error', 'post_json error: ' .. code)
                return server_internal_err_menu
            else
                engine:log('info', 'post_json result: \n' .. printTable(res, 0))
                response_id = res.response_id
                local ouput = res.output[1]
                engine:log("debug", "output: " .. ouput["content"])
                local json, err = engine:json_decode(ouput["content"])
                if err ~= nil then
                    engine:log('error', 'json_decode error: ' .. err)
                    return server_internal_err_menu
                else
                    engine:log("debug", "json: " .. printTable(json, 0))
                    local intent = json["intent"]
                    if intent == nil then
                        engine:log('error', 'json_decode error: intent is nil')
                        return server_internal_err_menu
                    else
                        engine:log("debug", "intent: " .. intent)
                        if intent == "tell_joke" then
                            return say_joke_menu
                        elseif intent == "tell_story" then
                            return say_story_menu
                        else
                            return not_find_intent_menu
                        end
                    end
                end
            end
        end
    end
}

local first_menu = {
    do_action = function()
        local asr, err = engine:say_and_detect(welcome, 0)
        if err ~= nil then
            engine:log('error', 'say_and_detect error: ' .. err)
        else
            engine:log('info', 'say_and_detect result: ' .. asr)
            if string.len(asr) == 0 then
                engine:log('error', 'say_and_detect result is empty')
                return not_find_intent_menu
                -- asr = "我想听故事"
            end
            big_model.input = asr
            if response_id ~= "" then
                big_model.response_id = response_id
            end
            local code, res, err = engine:post_json(big_mode_url, {}, big_model, 5000)
            if err ~= nil then
                engine:log('error', 'post_json error: ' .. err)
                return server_internal_err_menu
            else
                if code ~= 200 then
                    engine:log('error', 'post_json error: ' .. code)
                    return server_internal_err_menu
                else
                    engine:log('info', 'post_json result: \n' .. printTable(res, 0))
                    response_id = res.response_id
                    local ouput = res.output[1]
                    engine:log("debug", "output: " .. ouput["content"])
                    local json, err = engine:json_decode(ouput["content"])
                    if err ~= nil then
                        engine:log('error', 'json_decode error: ' .. err)
                        return server_internal_err_menu
                    else
                        engine:log("debug", "json: " .. printTable(json, 0))
                        local intent = json["intent"]
                        if intent == nil then
                            engine:log('error', 'json_decode error: intent is nil')
                            return server_internal_err_menu
                        else
                            engine:log("debug", "intent: " .. intent)
                            if intent == "tell_joke" then
                                return say_joke_menu
                            elseif intent == "tell_story" then
                                return say_story_menu
                            else
                                return not_find_intent_menu
                            end
                        end
                    end
                end
            end
        end
    end,

    next = function()
        return end_menu
    end
}

local menu = first_menu
while engine:is_ok() do
    local nxt_menu = menu:do_action()
    if nxt_menu == nil then
        engine:log('error', 'first_menu do_action error')
        break
    end
    menu = nxt_menu
end
