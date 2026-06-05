local local_package_path = freeswitch.getGlobalVariable("script_dir").."/?.lua"
package.path = package.path .. ";" .. local_package_path

local job_id = argv[1]
local task_id = argv[2]
local session_id = argv[3]
local display_number = argv[4]
local number = argv[5]
local robot_session_id = argv[6]
local robot_arg = argv[7]

local api = freeswitch.API();

local function NotifyResult(code, detail)
    local event = freeswitch.Event("custom", "cus_event::promise");
    event:addHeader("cus_event_job_id", job_id);
    event:addHeader("cus_event_code", code); -- 0 : success, 其他失败
    event:addHeader("cus_event_message", detail);

    event:fire(); 
    freeswitch.consoleLog("debug", event:serialize() .. "\n");
end

local function urlEncode(s)
    return string.gsub(s, "([^%w%.%-])", function(c) return string.format("%%%02X", string.byte(c)) end)
end

local function urlDecode(s)
    s = string.gsub(s, '%%(%x%x)', function(h) return string.char(tonumber(h, 16)) end)
    return s
end

local function format_var(kv_table)
    local var = ""
    if (type(kv_table) == "table") then 
        for k,v in pairs(kv_table) do
            if (k ~= nil and v ~= nil and type(v) == "string" and type(k) == "string" and k ~= "" and v ~= "") then
                if (k ~= "api_hangup_hook" and k ~= "api_on_answer" and k ~= "execute_on_media") then
                    v = urlEncode(urlDecode(v));
                end
                var = var .. k .. "=" .. v .. ","
            end
        end
    end
    if (var:sub(-1) == ',') then
        var = var:sub(1, -2)
    end
    return var
end

local function format_originator_string(variable, number)
    local originator_str = "originate";
    if (string.find(number, "$")) then
        originator_str = "expand originate ";
    end

    originator_str = originator_str .. "{" .. variable .. "}" .. number .. " &park";
    return originator_str;
end

local kv_table_robot = {};
kv_table_robot["origination_caller_id_number"] = display_number
kv_table_robot["task_id"] = task_id;
kv_table_robot["originator_codec"] = "PCMA"
kv_table_robot["origination_uuid"] = robot_session_id
kv_table_robot["raw_flowdata"] = robot_arg
local variable = format_var(kv_table_robot);
local ori_str = format_originator_string(variable, number)
freeswitch.consoleLog("debug", ori_str .. "\n");

local resp = freeswitch.API():executeString(ori_str);
if (resp == nil) then
    resp = "unknown";
end

freeswitch.consoleLog("debug", "originate result :" .. resp .. "\n");
if (string.find(resp, "+OK") == nil) then
    -- should notify failed
    NotifyResult(-2, resp);
    return ;
end

freeswitch.API():executeString("uuid_bridge " .. session_id .. " " .. robot_session_id);
NotifyResult(0, resp);