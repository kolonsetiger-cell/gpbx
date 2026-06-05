local local_package_path = freeswitch.getGlobalVariable("script_dir").."/?.lua"
package.path = package.path .. ";" .. local_package_path

local job_id = argv[1]
local task_id = argv[2]
local a_uuid = argv[3]
local a_caller = argv[4]
local a_callee = argv[5]
local a_way_type = argv[6]
local a_way = argv[7]
local b_uuid = argv[8]
local b_caller = argv[9]
local b_callee = argv[10]
local b_way_type = argv[11]
local b_way = argv[12]
local timeout = argv[13]

-- local api=freeswitch.API();

local function NotifyResult(code, detail)
    local event = freeswitch.Event("custom", "cus_event::promise");
    event:addHeader("cus_event_job_id", job_id);
    event:addHeader("cus_event_code", code); -- 0 : success, 其他失败
    event:addHeader("cus_event_message", detail);

    event:fire(); 
    freeswitch.consoleLog("debug", event:serialize() .. "\n");
end

local function NotifyCallStatus(number, status)
    local event = freeswitch.Event("custom", "cus_event::callstatus");
    event:addHeader("number", number);
    event:addHeader("status", status); -- 0 : success, 其他失败
    event:addHeader("time", os.time() * 1000000);

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

local function format_number(number, originate_type, originate_arg)
    if (originate_type == "gateway") then
        return "sofia/gateway/" .. originate_arg .. "/" .. number;
    end

    if (originate_type == "local") then
        return "user/" .. number;
    end

    if (originate_type == "ims") then
        return "sofia/external/" .. number .. "@" .. originate_arg
    end
    return "";
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

local caller = format_number(a_callee, a_way_type, a_way);
if (caller == "") then
    NotifyResult(-3, "Param Error");
    return 
end

local kv_table = {};
kv_table["origination_caller_id_number"] = a_caller
kv_table["origination_callee_id_number"] = a_callee
kv_table["effective_caller_id_number"] = a_caller
kv_table["effective_callee_id_number"] = a_callee

kv_table["task_id"] = task_id;
kv_table["originator_codec"] = "PCMA"
kv_table["origination_uuid"] = a_uuid
kv_table["originate_timeout"] = timeout
local variable = format_var(kv_table);

local function format_originator_string(variable, caller)
    local originator_str = "originate";
    if (string.find(caller, "$")) then
        originator_str = "expand originate ";
    end

    originator_str = originator_str .. "{" .. variable .. "}" .. caller .. " &park";
    return originator_str;
end

local ori_str = format_originator_string(variable, caller)
freeswitch.consoleLog("debug", ori_str .. "\n");

NotifyCallStatus(a_callee, "RING")
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

local callee = format_number(b_callee, b_way_type, b_way);
kv_table = {};
kv_table["origination_caller_id_number"] = b_caller
kv_table["origination_callee_id_number"] = b_callee
kv_table["effective_caller_id_number"] = b_caller
kv_table["effective_callee_id_number"] = b_callee
kv_table["task_id"] = task_id;
kv_table["originator_codec"] = "PCMA"
kv_table["origination_uuid"] = b_uuid
kv_table["originate_timeout"] = timeout
variable = format_var(kv_table);
ori_str = format_originator_string(variable, callee)

NotifyCallStatus(a_callee, "RINGBACK")
local resp = freeswitch.API():executeString(ori_str);
if (resp == nil) then
    resp = "unknown";
end

freeswitch.consoleLog("debug", "originate result :" .. resp .. "\n");
if (string.find(resp, "+OK") == nil) then
    -- should notify failed
    freeswitch.API():executeString("uuid_kill " .. a_uuid)
    NotifyResult(-2, resp);
    return ;
end

freeswitch.API():executeString("uuid_bridge " .. a_uuid .. " " .. b_uuid)
NotifyResult(0, resp);
NotifyCallStatus(a_callee, "INCALL")