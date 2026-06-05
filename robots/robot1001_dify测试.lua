-- 机器人1001  使用 dify 测试

local asr, err = engine:say_and_detect("这里是配对平台，您可以说出您的条件，选择您的理想类型。", 5000)
if err ~= nil then
    engine:log('info', 'err = ' .. err)
end

local last_llm_res = ""
while engine:is_ok() do
::wait_asr::
    if asr == nil or #asr == 0 then
        engine:sleep(1000)
        asr, _ = engine:say_and_detect("这里是配对平台，您可以说出您的条件，选择您的理想类型。", 5000)
        goto wait_asr
    end
    -- engine:enable_asr(true)
    -- engine:enable_asr(false)
::wait_llm::
    local res, err = engine:llm_say_raw("", asr, -1)
    if err ~= nil then
        engine:log('info', 'err = ' .. err)
        asr, _ = engine:say_and_detect("抱歉，系统异常！这里是配对平台，您可以说出您的条件，选择您的理想类型。", 5000)
        goto wait_asr
    end

    if res == nil or #res > 64 or #res == 0 then
        if last_llm_res ~= "" then
            asr, _ = engine:say_and_detect(last_llm_res, 5000)
            goto wait_llm
        else
            asr, _ = engine:say_and_detect("抱歉，我不清楚您想说什么！这里是配对平台，您可以说出您的条件，选择您的理想类型。", 5000)
            goto wait_asr
        end
    end

---@diagnostic disable-next-line: cast-local-type
    last_llm_res = res
    asr, _ = engine:say_and_detect(res, 5000)
    goto wait_llm
end

engine:log('info', 'Robot End')
