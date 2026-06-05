package ai

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

	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"
)

type LuaEngine struct {
	lu         *lua.LState
	is_ok      bool
	file       string
	exit       chan bool
	mu         sync.Mutex
	asr        []string
	enable_asr bool
	session_id string
	ai         ai_vendor
}

func (l *LuaEngine) IsOk() bool {
	return l.is_ok
}

func (l *LuaEngine) setAiVendor(ai ai_vendor) {
	l.ai = ai
}

func (l *LuaEngine) setAsr(asr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(asr) == 0 {
		return
	}
	if l.enable_asr {
		l.asr = append(l.asr, asr)
	}
}

func (l *LuaEngine) LLMSayOutJson(prompt, text string, timeout_ms int) (map[string]interface{}, error) {
	if l.ai == nil {
		return nil, nil
	}
	out, err := l.ai.Say(prompt, text, timeout_ms)
	if err != nil {
		return nil, err
	}
	var json_out map[string]any
	err = json.Unmarshal([]byte(out), &json_out)
	if err != nil {
		return nil, err
	}
	return json_out, nil
}

func (l *LuaEngine) LLMSayOutRaw(prompt, text string, timeout_ms int) (string, error) {
	if l.ai == nil {
		return "", nil
	}
	out, err := l.ai.Say(prompt, text, timeout_ms)
	if err != nil {
		return "", err
	}
	return out, nil
}
func (l *LuaEngine) EnableAsr(enable_asr bool) {
	l.enable_asr = enable_asr
}

func (l *LuaEngine) GetOneAsr() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.asr) == 0 {
		return ""
	}
	ret := l.asr[len(l.asr)-1]
	l.asr = []string{}
	return ret
}

func (l *LuaEngine) Sleep(ms int) {
	sleep_ms := 0
	for sleep_ms < ms && l.IsOk() {
		time.Sleep(time.Millisecond * time.Duration(10))
		sleep_ms += 10
	}
}

func (l *LuaEngine) GetSomeAsr() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := l.asr
	l.asr = []string{}
	return ret
}

func (l *LuaEngine) WaitAsr(timeout_ms int) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.EnableAsr(true)
	var asr string
	is_timeout := false
	to_inst := time.NewTimer(time.Millisecond * time.Duration(timeout_ms))
	defer to_inst.Stop()
	go func() {
		<-to_inst.C
		is_timeout = true
	}()
	for !is_timeout {
		asrs := l.GetSomeAsr()
		if len(asrs) != 0 {
			for _, res := range asrs {
				asr += res
			}
			break
		}
		time.Sleep(time.Millisecond * time.Duration(100))
	}
	l.EnableAsr(false)
	return asr
}

func (l *LuaEngine) Playback(file string) error {
	uid := uuid.New()
	result := pbx.RobotPlayback(uid.String(), l.session_id, pbx.Playback{
		Batch: []pbx.Content{
			{
				Type:    pbx.CONTENT_TYPE_FILE,
				Content: file,
			},
		},
		IsBreak:   true,
		IsHangup:  false,
		IsAsync:   true,
		LoopCount: 1,
	})
	if result.Code != 0 {
		return errors.New(strconv.Itoa(result.Code))
	}
	return nil
}

func (l *LuaEngine) PlaybackSync(file string) error {
	uid := uuid.New()
	result := pbx.RobotPlayback(uid.String(), l.session_id, pbx.Playback{
		Batch: []pbx.Content{
			{
				Type:    pbx.CONTENT_TYPE_FILE,
				Content: file,
			},
		},
		IsBreak:   true,
		IsHangup:  false,
		IsAsync:   false,
		LoopCount: 1,
	})
	if result.Code != 0 {
		return errors.New(strconv.Itoa(result.Code))
	}
	return nil
}

func (l *LuaEngine) Say(text string) error {
	uid := uuid.New()
	result := pbx.RobotPlayback(uid.String(), l.session_id, pbx.Playback{
		Batch: []pbx.Content{
			{
				Type:    pbx.CONTENT_TYPE_TTS,
				Content: text,
			},
		},
		IsBreak:   true,
		IsHangup:  false,
		IsAsync:   true,
		LoopCount: 1,
	})
	if result.Code != 0 {
		return errors.New(strconv.Itoa(result.Code))
	}
	return nil
}

func (l *LuaEngine) SaySync(text string) error {
	uid := uuid.New()
	result := pbx.RobotPlayback(uid.String(), l.session_id, pbx.Playback{
		Batch: []pbx.Content{
			{
				Type:    pbx.CONTENT_TYPE_TTS,
				Content: text,
			},
		},
		IsBreak:   true,
		IsHangup:  false,
		IsAsync:   false,
		LoopCount: 1,
	})
	if result.Code != 0 {
		return errors.New(strconv.Itoa(result.Code))
	}
	return nil
}

func (l *LuaEngine) SayAndDetect(text string, timeout_ms int) (string, error) {
	play_end := make(chan event.Result)
	go func() {
		uid := uuid.New()
		result := pbx.RobotPlaybackWithTimeout(uid.String(), l.session_id, pbx.Playback{
			Batch: []pbx.Content{
				{
					Type:    pbx.CONTENT_TYPE_TTS,
					Content: text,
				},
			},
			IsBreak:   true,
			IsHangup:  false,
			IsAsync:   false,
			LoopCount: 1,
		}, -1)
		play_end <- result
	}()

	// if result.Code != 0 {
	// 	return "", errors.New(strconv.Itoa(result.Code))
	// }
	asr_end := make(chan string)
	l.EnableAsr(true)
	is_play_end := false
	go func() {
		// 等待 ASR 数据
		sleep_now := 0

		// 至少等待500ms后才获取 ASR， 不能一播放就被用户给打断
		for l.IsOk() && sleep_now < 500 {
			time.Sleep(time.Millisecond * 10)
			sleep_now += 10
		}
		res := ""
		sleep_now = 0
		for l.IsOk() && sleep_now < timeout_ms {
			asr := l.GetSomeAsr()
			if len(asr) > 0 {
				// l.EnableAsr(false)
				// l.Break()
				var str strings.Builder
				for _, v := range asr {
					_, _ = str.WriteString(v + ",")
				}
				_, _ = str.WriteString("\n")
				res = str.String()
				break
			}
			time.Sleep(time.Millisecond * 10)
			if is_play_end {
				sleep_now += 10
			}
		}
		asr_end <- res
	}()
	var result event.Result
	var asr_result string
	select {
	case result = <-play_end:
		is_play_end = true
	case asr_result = <-asr_end:
	}
	if is_play_end {
		asr_result = <-asr_end
		l.EnableAsr(false)
		close(asr_end)
		close(play_end)
		if result.Code != 0 {
			return "", errors.New(strconv.Itoa(result.Code))
		}
		return asr_result, nil
	}
	if !is_play_end {
		l.Break()
	}
	<-play_end
	close(asr_end)
	close(play_end)
	defaultLogger.Info(ThisModule, "<%v> SayAndDetect %v, asr: %v", l.session_id, text, asr_result)
	return asr_result, nil
}

func (l *LuaEngine) SayAndDetectWithCallback(text string, timeout_ms int, callback func(asr string) bool) error {
	play_end := make(chan event.Result)
	go func() {
		uid := uuid.New()
		result := pbx.RobotPlaybackWithTimeout(uid.String(), l.session_id, pbx.Playback{
			Batch: []pbx.Content{
				{
					Type:    pbx.CONTENT_TYPE_TTS,
					Content: text,
				},
			},
			IsBreak:   true,
			IsHangup:  false,
			IsAsync:   false,
			LoopCount: 1,
		}, -1)
		play_end <- result
	}()

	// if result.Code != 0 {
	// 	return "", errors.New(strconv.Itoa(result.Code))
	// }
	asr_end := make(chan string)
	l.EnableAsr(true)
	is_play_end := false
	go func() {
		// 等待 ASR 数据
		sleep_now := 0

		// 至少等待500ms后才获取 ASR， 不能一播放就被用户给打断
		for l.IsOk() && sleep_now < 500 {
			time.Sleep(time.Millisecond * 10)
			sleep_now += 10
		}
		res := ""
		sleep_now = 0
		for l.IsOk() && sleep_now < timeout_ms {
			asr := l.GetSomeAsr()
			if len(asr) > 0 {
				// l.EnableAsr(false)
				// l.Break()
				var str strings.Builder
				for _, v := range asr {
					_, _ = str.WriteString(v + ",")
				}
				_, _ = str.WriteString("\n")
				res = str.String()
				if callback != nil && callback(res) {
					break
				}
			}
			time.Sleep(time.Millisecond * 10)
			if is_play_end {
				sleep_now += 10
			}
		}
		asr_end <- res
	}()
	var result event.Result
	var asr_result string
	select {
	case result = <-play_end:
		is_play_end = true
	case asr_result = <-asr_end:
	}
	if is_play_end {
		<-asr_end
		l.EnableAsr(false)
		close(asr_end)
		close(play_end)
		if result.Code != 0 {
			return errors.New(strconv.Itoa(result.Code))
		}
		return nil
	}
	if !is_play_end {
		l.Break()
	}
	<-play_end
	close(asr_end)
	close(play_end)
	defaultLogger.Info(ThisModule, "<%v> SayAndDetect %v, asr: %v", l.session_id, text, asr_result)
	return nil
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
	pbx.RobotBreak(l.session_id)
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
		l.lu.SetField(table, "llm_say_json", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			prompt := ls.CheckString(2)
			text := ls.CheckString(3)
			timeout_ms := ls.CheckNumber(4)
			out, err := l.LLMSayOutJson(prompt, text, int(timeout_ms))
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> LLMSayOutJson %v, Result: <%v:%v>", l.file, prompt, err)
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			dstTbl, err := any_to_table(ls, out)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> any_to_table %v, Result: <%v:%v>", l.file, out, err)
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			ls.Push(dstTbl)
			ls.Push(lua.LNil)
			return 2
		}))
		l.lu.SetField(table, "llm_say_raw", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			prompt := ls.CheckString(2)
			text := ls.CheckString(3)
			timeout_ms := ls.CheckNumber(4)
			out, err := l.LLMSayOutRaw(prompt, text, int(timeout_ms))
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> LLMSayOutRaw %v, Result: <%v:%v>", l.file, prompt, err)
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}

			ls.Push(lua.LString(out))
			ls.Push(lua.LNil)
			return 2
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
				defaultLogger.Error(ThisModule, "<%v> Playback %v, Result: <%v:%v>", l.file, file, err)
				ls.Push(lua.LString(err.Error()))
				return 1
			}
			ls.Push(lua.LString("success"))
			return 1
		}))
		l.lu.SetField(table, "playback_sync", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			file := ls.CheckString(2)
			err := l.PlaybackSync(file)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> PlaybackSync %v, Result: <%v:%v>", l.session_id, file, err)
				ls.Push(lua.LString(err.Error()))
				return 1
			}
			ls.Push(lua.LString("success"))
			return 1
		}))
		l.lu.SetField(table, "say", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			text := ls.CheckString(2)
			err := l.Say(text)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> Say %v, Result: <%v:%v>", l.file, text, err)
				ls.Push(lua.LString(err.Error()))
				return 1
			}
			ls.Push(lua.LString("success"))
			return 1
		}))
		l.lu.SetField(table, "say_sync", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			text := ls.CheckString(2)
			err := l.SaySync(text)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> SaySync %v, Result: <%v:%v>", l.file, text, err)
				ls.Push(lua.LString(err.Error()))
				return 1
			}
			ls.Push(lua.LString("success"))
			return 1
		}))
		l.lu.SetField(table, "say_and_detect", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			text := ls.CheckString(2)
			timeout_ms := ls.CheckInt(3)
			asr, err := l.SayAndDetect(text, timeout_ms)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> SayAndDetect %v, Result: <%v:%v>", l.file, text, err)
				ls.Push(lua.LString(""))
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			ls.Push(lua.LString(asr))
			ls.Push(lua.LNil)
			return 2
		}))
		l.lu.SetField(table, "say_and_detect_with_callback", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			text := ls.CheckString(2)
			timeout_ms := ls.CheckInt(3)
			fn := ls.CheckFunction(4)
			callback := func(asr string) bool {
				l.mu.Lock()
				defer l.mu.Unlock()
				// --------------------------
				// 【核心：Go 调用 Lua 函数】
				// --------------------------
				// 1. 压入函数
				l.lu.Push(fn)
				// 2. 压入参数（asr 结果）
				l.lu.Push(lua.LString(asr))
				// 3. 调用：1个参数，1个返回值
				l.lu.Call(1, 1)
				// 4. 获取返回值（栈顶）
				ret := l.lu.ToBool(-1)
				l.lu.Pop(1) // 清理栈
				return ret
			}
			err := l.SayAndDetectWithCallback(text, timeout_ms, callback)
			if err != nil {
				defaultLogger.Error(ThisModule, "<%v> SayAndDetectWithCallback %v, Result: <%v:%v>", l.file, text, err)
				ls.Push(lua.LString(err.Error()))
				return 1
			}
			ls.Push(lua.LNil)
			return 1
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
		l.lu.SetField(table, "enable_asr", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			enable_asr := ls.CheckBool(2)
			l.EnableAsr(enable_asr)
			return 0
		}))
		l.lu.SetField(table, "sleep", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			ms := ls.CheckInt(2)
			l.Sleep(ms)
			return 0
		}))
		l.lu.SetField(table, "get_asr", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			asr := l.GetOneAsr()
			ls.Push(lua.LString(asr))
			return 1
		}))
		l.lu.SetField(table, "get_some_asr", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			asr := l.GetSomeAsr()
			v_table := l.lu.NewTable()
			for i, s := range asr {
				// Lua 数组下标从 1 开始
				l.lu.SetTable(v_table, lua.LNumber(i+1), lua.LString(s))
			}
			ls.Push(v_table)
			return 1
		}))
		l.lu.SetField(table, "wait_asr", l.lu.NewFunction(func(ls *lua.LState) int {
			ls.CheckTable(1)
			ms := ls.CheckInt(2)
			asr := l.WaitAsr(ms)
			ls.Push(lua.LString(asr))
			return 1
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
