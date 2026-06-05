-- table, err := engine:get_message()
-- table: 
	-- from_username string
	-- from_id       int64
	-- chat_id       int64
	-- text          string
function string.trim(s)
    -- 匹配开头空白+结尾空白，替换为空字符串
    return s:gsub("^%s+", ""):gsub("%s+$", "")
end

local call_id = nil
while true do
    local msg, err = engine:get_message()
    if err ~= nil then
        engine:log("error", "get_message error: " .. err)
        break
    end
    if msg.event == "telegram" then
        call_id = msg.chat_id
        if msg.text == "/start" then
            engine:send_message(msg.chat_id, "/call <number>")
        elseif string.sub(msg.text, 1, 5) == "/call" then
            local number = string.trim(string.sub(msg.text, 6))
            if number == "" then
                engine:send_message(msg.chat_id, "number is empty")
            else
                engine:send_message(msg.chat_id, "start call number " .. number)
                local err = engine:call(number)
                if err ~= nil then
                    engine:send_message(msg.chat_id, "call error: " .. err)
                else
                    engine:send_message(msg.chat_id, "call success")
                end
            end
        elseif string.sub(msg.text, 1, 5) == "/test" then
            local number = string.trim(string.sub(msg.text, 6))
            if number == "" then
                engine:send_message(msg.chat_id, "number is empty")
            else
                engine:send_message(msg.chat_id, "start call number " .. number)
                local err = engine:test(number)
                if err ~= nil then
                    engine:send_message(msg.chat_id, "call error: " .. err)
                else
                    engine:send_message(msg.chat_id, "call success")
                end
            end
        else
            engine:send_message(msg.chat_id, "/start          --> show your self info")
            engine:send_message(msg.chat_id, "/call <number>  --> call a number")
            engine:send_message(msg.chat_id, "/test <number>  --> test a number")
        end
    elseif msg.event == "ensure" then
        engine:send_message(call_id, "🚀🚀🚀🚀🚀🚀🚀🚀🚀🚀🚀🚀")
    elseif msg.event == "answer" then
        engine:send_message(call_id, "call answer")
    elseif msg.event == "hangup" then
        engine:send_message(call_id, "call hangup")
    elseif msg.event == "dtmf" then
        engine:send_message(call_id, "digit:" .. msg.data)
    end
end

engine:log("info", "lua end")