package ivr

/**
 * 1. lua 支持播放功能
 * 2. lua 支持设置是否接收 ASR 数据功能
 * 3. lua 支持
 */

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	lua "github.com/yuin/gopher-lua"
)

type LuaEngine struct {
	lu             *lua.LState
	is_ok          bool
	file           string
	exit           chan bool
	mu             sync.Mutex
	asr            []string
	enable_asr     bool
	session_id     string
	dtmf_que       []string
	dtmf_mu        sync.Mutex
	dtmf_enable    bool
	event_callback func(string, string)
}

func (l *LuaEngine) pushDtmf(dtmf string) {
	l.dtmf_mu.Lock()
	defer l.dtmf_mu.Unlock()
	if l.dtmf_enable {
		l.dtmf_que = append(l.dtmf_que, dtmf)
		if l.event_callback != nil {
			l.event_callback("dtmf", dtmf)
		}
	}
}

func (l *LuaEngine) EnableDtmf(enable bool) {
	l.dtmf_mu.Lock()
	defer l.dtmf_mu.Unlock()
	l.dtmf_enable = enable
	l.dtmf_que = l.dtmf_que[:0]
}

func (l *LuaEngine) GetDtmfs() string {
	var ret string
	l.dtmf_mu.Lock()
	defer l.dtmf_mu.Unlock()
	ret = strings.Join(l.dtmf_que, "")
	l.dtmf_que = l.dtmf_que[:0]
	return ret
}

func (l *LuaEngine) IsOk() bool {
	return l.is_ok
}

func (l *LuaEngine) Sleep(ms int) {
	sleep_ms := 0
	for sleep_ms < ms && l.IsOk() {
		time.Sleep(time.Millisecond * time.Duration(10))
		sleep_ms += 10
	}
}

func (l *LuaEngine) Playback(file string) error {
	result := pbx.SessionPlayback(l.session_id, file, 120)
	if result.Code != 0 {
		return errors.New(strconv.Itoa(result.Code))
	}
	return nil
}

func (l *LuaEngine) PlayAndGetDtmf(file string, hope_dtmf string, timeout int) string {
	play_done := false
	go func() {
		pbx.SessionPlayback(l.session_id, file, timeout/1000)
		play_done = true
	}()
	l.EnableDtmf(true)
	defer l.EnableDtmf(false)
	step := 50
	cur := 0
	for l.IsOk() && cur < timeout {
		dtmf := l.GetDtmfs()
		// defaultLogger.Info(ThisModule, "<%v> PlayAndGetDtmf dtmf: %v", l.session_id, dtmf)
		if len(dtmf) > 0 {
			for _, d := range dtmf {
				if strings.Contains(hope_dtmf, string(d)) {
					if !play_done {
						pbx.ApiBreak(l.session_id)
					}
					return string(d)
				}
			}
		}

		if play_done {
			cur += step
		}
		time.Sleep(time.Millisecond * time.Duration(step))
	}
	return ""
}

func (l *LuaEngine) PlayAndGetDtmfs(file string, hope_len int, timeout int) string {
	play_done := false
	go func() {
		pbx.SessionPlayback(l.session_id, file, timeout/1000)
		play_done = true
	}()
	l.EnableDtmf(true)
	defer l.EnableDtmf(false)
	step := 50
	cur := 0
	dtmfs := ""
	for l.IsOk() && cur < timeout {
		dtmfs += l.GetDtmfs()
		if len(dtmfs) >= hope_len {
			if !play_done {
				pbx.ApiBreak(l.session_id)
			}
			return dtmfs[:hope_len]
		}

		if play_done {
			cur += step
		}
		time.Sleep(time.Millisecond * time.Duration(step))
	}
	return ""
}

func (l *LuaEngine) PlayAndRequest(file string, url string, body map[string]any, timeout int) (map[string]any, error) {
	pbx.SessionPlaybackAsync(l.session_id, file)
	_, response, err := l.PostJson(url, nil, body, timeout)
	l.Break()
	if err != nil {
		return nil, err
	}
	return response, nil
}
func (l *LuaEngine) PostJson(url string, header map[string]string, body map[string]any, timeout_ms int) (int, map[string]any, error) {
	client := &http.Client{
		Timeout: time.Millisecond * time.Duration(timeout_ms), // 10秒超时
	}
	defaultLogger.Info(ThisModule, "<%v> PostJson %v, body: %v timeout_ms: %v", l.session_id, url, body, timeout_ms)
	body_bytes, _ := json.Marshal(body)
	req, _ := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer(body_bytes),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Golang GPBX")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, nil, errors.New("PostJson failed, status code " + strconv.Itoa(resp.StatusCode))
	}
	body_bytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	var response map[string]any
	err = json.Unmarshal(body_bytes, &response)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, response, nil
}

func (l *LuaEngine) Break() {
	pbx.ApiBreak(l.session_id)
}

func (l *LuaEngine) Hangup() {
	pbx.HangupCall(l.session_id)
}

func (l *LuaEngine) Log(level string, log string) {
	switch level {
	case "debug":
		defaultLogger.Debug(ThisModule, "<%v> %v", l.session_id, log)
	case "info":
		defaultLogger.Info(ThisModule, "<%v> %v", l.session_id, log)
	case "error":
		defaultLogger.Error(ThisModule, "<%v> %v", l.session_id, log)
	case "warn":
		defaultLogger.Warn(ThisModule, "<%v> %v", l.session_id, log)
	default:
		defaultLogger.Debug(ThisModule, "<%v> %v", l.session_id, log)
	}
}

func (l *LuaEngine) JsonEncode(src any) string {
	json_str, _ := json.Marshal(src)
	return string(json_str)
}

func (l *LuaEngine) JsonDecode(src string) (any, error) {
	var dst any
	err := json.Unmarshal([]byte(src), &dst)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (l *LuaEngine) SetCallback(f_callback func(string, string)) {
	l.event_callback = f_callback
}

func table_to_map_string(tbl *lua.LTable) map[string]string {
	result := make(map[string]string)
	// 遍历 Lua 表
	tbl.ForEach(func(key lua.LValue, val lua.LValue) {
		keyStr := lua.LVAsString(key) // 转字符串 key

		// 根据类型取值
		switch val.Type() {
		case lua.LTString:
			result[keyStr] = lua.LVAsString(val)
		}
	})

	return result
}

func table_to_array_any(tbl *lua.LTable) []any {
	result := make([]any, 0)
	// 遍历 Lua 表
	tbl.ForEach(func(key lua.LValue, val lua.LValue) {
		// 根据类型取值
		switch val.Type() {
		case lua.LTString:
			result = append(result, lua.LVAsString(val))
		case lua.LTNumber:
			result = append(result, float64(lua.LVAsNumber(val)))
		case lua.LTBool:
			result = append(result, lua.LVAsBool(val))
		case lua.LTTable:
			if is_array(val.(*lua.LTable)) {
				result = append(result, table_to_array_any(val.(*lua.LTable)))
			} else {
				result = append(result, table_to_map_any(val.(*lua.LTable)))
			}
		default:
		}
	})

	return result
}

func table_to_map_any(tbl *lua.LTable) map[string]any {
	result := make(map[string]any)
	// 遍历 Lua 表
	tbl.ForEach(func(key lua.LValue, val lua.LValue) {
		keyStr := lua.LVAsString(key) // 转字符串 key
		// 根据类型取值
		switch val.Type() {
		case lua.LTString:
			result[keyStr] = lua.LVAsString(val)
		case lua.LTNumber:
			result[keyStr] = int(lua.LVAsNumber(val))
		case lua.LTBool:
			result[keyStr] = lua.LVAsBool(val)
		case lua.LTTable:
			if is_array(val.(*lua.LTable)) {
				result[keyStr] = table_to_array_any(val.(*lua.LTable))
			} else {
				result[keyStr] = table_to_map_any(val.(*lua.LTable))
			}
		default:
			result[keyStr] = nil
		}
	})

	return result
}

func is_array(tbl *lua.LTable) bool {
	is_array := true
	tbl.ForEach(func(key lua.LValue, val lua.LValue) {
		if key.Type() != lua.LTNumber { // 如果 key 是 number， 说明是数组
			is_array = false
		}
	})
	return is_array
}

func array_any_to_table(L *lua.LState, arr []any) *lua.LTable {
	tbl := L.NewTable()
	for i, v := range arr {
		switch val := v.(type) {
		case string:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LString(val))
		case int:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LNumber(float64(val)))
		case bool:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LBool(val))
		case map[string]any:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), map_any_to_table(L, val))
		case []any:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), array_any_to_table(L, val))
		default:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LNil)
		}
	}
	return tbl
}

func map_any_to_table(L *lua.LState, m map[string]any) *lua.LTable {
	tbl := L.NewTable()
	for k, v := range m {

		switch val := v.(type) {
		case string:
			L.SetField(tbl, k, lua.LString(val))
		case int:
			L.SetField(tbl, k, lua.LNumber(float64(val)))
		case float64:
			L.SetField(tbl, k, lua.LNumber(val))
		case bool:
			L.SetField(tbl, k, lua.LBool(val))
		case map[string]any:
			L.SetField(tbl, k, map_any_to_table(L, val))
		case []any:
			L.SetField(tbl, k, array_any_to_table(L, val))
		default:
			L.SetField(tbl, k, lua.LNil)
			defaultLogger.Debug(ThisModule, "%v:%v:%v", k, v, reflect.TypeOf(val))
		}
	}
	return tbl
}

func table_to_any(tbl *lua.LTable) any {
	if is_array(tbl) {
		return table_to_array_any(tbl)
	}

	return table_to_map_any(tbl)
}

func any_to_table(L *lua.LState, v any) (*lua.LTable, error) {
	switch vv := v.(type) {
	case map[string]any:
		return map_any_to_table(L, vv), nil
	case []any:
		return array_any_to_table(L, vv), nil
	default:
		return nil, fmt.Errorf("any_to_table: %v", reflect.TypeOf(vv))
	}
}

func (l *LuaEngine) do() {
	go func() {
		table := l.lu.NewTable()
		l.lu.SetGlobal("engine", table)
		l.lu.SetField(table, "get_uuid", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			ls.Push(lua.LString(l.session_id))
			return 1
		}))

		l.lu.SetField(table, "set_callback", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			fn := ls.CheckFunction(2)
			callback := func(event string, data string) {
				l.mu.Lock()
				defer l.mu.Unlock()
				// --------------------------
				// 【核心：Go 调用 Lua 函数】
				// --------------------------
				// 1. 压入函数
				l.lu.Push(fn)
				// 2. 压入参数（asr 结果）
				l.lu.Push(lua.LString(event))
				l.lu.Push(lua.LString(data))
				// 3. 调用：1个参数，1个返回值
				l.lu.Call(2, 0)
			}
			l.SetCallback(callback)
			return 0
		}))
		l.lu.SetField(table, "json_encode", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			src := ls.CheckTable(2)
			dst := table_to_any(src)
			ls.Push(lua.LString(l.JsonEncode(dst)))
			return 1
		}))
		l.lu.SetField(table, "json_decode", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			src := ls.CheckString(2)
			dst, err := l.JsonDecode(src)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> JsonDecode %v, Result: <%v:%v>", l.file, src, err)
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			dstTbl, err := any_to_table(ls, dst)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> JsonDecode %v, Result: <%v:%v>", l.file, src, err)
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			ls.Push(dstTbl)
			ls.Push(lua.LNil)
			return 2
		}))
		l.lu.SetField(table, "post_json", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			url := ls.CheckString(2)
			header := ls.CheckTable(3)
			body := ls.CheckTable(4)
			timeout_ms := ls.CheckNumber(5)
			code, response, err := l.PostJson(url, table_to_map_string(header), table_to_map_any(body), int(timeout_ms))
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> PostJson %v, Result: <%v:%v>", l.file, url, err)
				ls.Push(lua.LNumber(code))
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 3
			}
			if code != 200 {
				defaultLogger.Error(ThisModule, "<%v> PostJson %v, Result: <%v:%v>", l.file, url, code, err)
				ls.Push(lua.LNumber(code))
				ls.Push(lua.LNil)
				ls.Push(lua.LNil)
				return 3
			}
			ls.Push(lua.LNumber(code))
			ls.Push(map_any_to_table(ls, response))
			ls.Push(lua.LNil)
			return 3
		}))
		l.lu.SetField(table, "log", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			level := ls.CheckString(2)
			log := ls.CheckString(3)
			l.Log(level, log)
			return 0
		}))
		l.lu.SetField(table, "hangup", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			l.Hangup()
			return 0
		}))
		l.lu.SetField(table, "playback", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			file := ls.CheckString(2)
			err := l.Playback(file)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> Playback %v, Result: <%v:%v>", l.session_id, file, err)
				ls.Push(lua.LString(err.Error()))
				return 1
			}
			ls.Push(lua.LString("success"))
			return 1
		}))
		l.lu.SetField(table, "play_and_get_digit", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			file := ls.CheckString(2)
			hope_dtmf := ls.CheckString(3)
			timeout_ms := ls.CheckNumber(4)
			ret := l.PlayAndGetDtmf(file, hope_dtmf, int(timeout_ms))
			ls.Push(lua.LString(ret))
			return 1
		}))
		l.lu.SetField(table, "play_and_get_digits", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			file := ls.CheckString(2)
			hope_len := ls.CheckNumber(3)
			timeout_ms := ls.CheckNumber(4)
			ret := l.PlayAndGetDtmfs(file, int(hope_len), int(timeout_ms))
			ls.Push(lua.LString(ret))
			return 1
		}))
		l.lu.SetField(table, "play_and_request_post", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			file := ls.CheckString(2)
			url := ls.CheckString(3)
			body := ls.CheckTable(4)
			timeout_ms := ls.CheckNumber(5)
			response, err := l.PlayAndRequest(file, url, table_to_map_any(body), int(timeout_ms))
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> PlayAndRequest %v, Result: <%v>", l.session_id, file, err)
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			defaultLogger.Info(ThisModule, "<%v> PlayAndRequest %v, Result: <%v>", l.session_id, file, response)
			ls.Push(map_any_to_table(ls, response))
			ls.Push(lua.LNil)
			return 2
		}))
		l.lu.SetField(table, "break", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			l.Break()
			return 0
		}))
		l.lu.SetField(table, "is_ok", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			ls.Push(lua.LBool(l.IsOk()))
			return 1
		}))
		l.lu.SetField(table, "sleep", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			ms := ls.CheckInt(2)
			l.Sleep(ms)
			return 0
		}))

		err := l.lu.DoFile(l.file)
		if err != nil {
			defaultLogger.Error(ThisModule, "<%v> DoFile %v, Result: <%v:%v>", l.file, err)
		}
		if l.is_ok {
			l.Hangup()
		}
		l.is_ok = false
		l.lu.Close()
		l.exit <- true
		close(l.exit)
		defaultLogger.Info(ThisModule, "<%v> LuaEngine Close", l.file)
	}()
}

func (l *LuaEngine) Close() {
	l.is_ok = false
	<-l.exit
}

func NewLuaEngine(file string, session_id string) *LuaEngine {
	return &LuaEngine{
		lu:         lua.NewState(),
		is_ok:      true,
		file:       file,
		enable_asr: false,
		exit:       make(chan bool),
		session_id: session_id,
	}
}
