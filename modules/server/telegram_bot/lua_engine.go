package telegram_bot

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	lua "github.com/yuin/gopher-lua"
)

// type Message struct {
// 	from_username string
// 	from_id       int64
// 	chat_id       int64
// 	text          string
// }

// LuaEngine 向 Lua 脚本暴露 Telegram Bot API
type LuaEngine struct {
	ctx          context.Context
	bot          *tgbot.Bot
	update       *tgmodels.Update
	lu           *lua.LState
	file         string
	exit         chan bool
	isOk         bool
	callback     EventCallback
	msg_que      chan *tgmodels.Update
	sess_msg_que chan map[string]any
	user         *tgmodels.User
}

// NewLuaEngine 创建 Lua 引擎实例
func NewLuaEngine(ctx context.Context, bot *tgbot.Bot, update *tgmodels.Update, scriptPath string, callback EventCallback) *LuaEngine {
	return &LuaEngine{
		ctx:          ctx,
		bot:          bot,
		update:       update,
		file:         scriptPath,
		exit:         make(chan bool, 1),
		isOk:         true,
		msg_que:      make(chan *tgmodels.Update, 100),
		sess_msg_que: make(chan map[string]any),
		callback:     callback,
		user:         update.Message.From,
	}
}

func (e *LuaEngine) OnMessage(update *tgmodels.Update) {
	defer func() {
		// 捕获异常
		_ = recover()
	}()
	e.msg_que <- update
}

func (e *LuaEngine) OnSessionMessage(msg map[string]any) {
	defer func() {
		// 捕获异常
		_ = recover()
	}()
	e.sess_msg_que <- msg
}
func (e *LuaEngine) GetMessage() (map[string]any, error) {
	timer := time.NewTicker(50 * time.Millisecond)
	defer timer.Stop()
	for e.isOk {
		select {
		case update := <-e.msg_que:
			return map[string]any{
				"from_username": update.Message.From.Username,
				"from_id":       update.Message.From.ID,
				"chat_id":       update.Message.Chat.ID,
				"text":          update.Message.Text,
				"event":         "telegram",
			}, nil
		case msg := <-e.sess_msg_que:
			return msg, nil
		case <-timer.C:
			// 超时后继续循环，检查 e.isOk 状态
			continue
		}
	}
	return nil, fmt.Errorf("Closed")
}

func (e *LuaEngine) Call(number string) error {
	if e.callback != nil {
		return e.callback(EVENT_CALLBACK_CALL, number)
	}
	return fmt.Errorf("system error, callback is nil")
}

func (e *LuaEngine) Test(number string) error {
	if e.callback != nil {
		return e.callback(EVENT_CALLBACK_TEST, number)
	}
	return fmt.Errorf("system error, callback is nil")
}

func array_any_to_table(L *lua.LState, arr []any) *lua.LTable {
	tbl := L.NewTable()
	for i, v := range arr {
		switch val := v.(type) {
		case string:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LString(val))
		case int:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LNumber(float64(val)))
		case int64:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LNumber(float64(val)))
		case float64:
			L.SetTable(tbl, lua.LNumber(float64(i+1)), lua.LNumber(val))
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
		case int64:
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

// Run 启动 Lua 引擎，注册 API 并执行脚本
func (e *LuaEngine) Run() {
	defer func() {
		e.isOk = false
		e.exit <- true
		close(e.exit)
		if e.callback != nil {
			e.callback(EVENT_CALLBACK_CLOSE, nil)
			e.callback = nil
		}
		if e.msg_que != nil {
			close(e.msg_que)
		}
		defaultLogger.Info(ThisModule, "LuaEngine closed, script: %s", e.file)
	}()

	e.lu = lua.NewState()
	defer e.lu.Close()

	// 注册 telegram 全局表
	telegramTable := e.lu.NewTable()
	e.lu.SetGlobal("engine", telegramTable)

	e.lu.SetField(telegramTable, "send_message", e.lu.NewFunction(func(ls *lua.LState) int {
		chatID := ls.Get(2)
		text := ls.CheckString(3)

		var cid any
		switch v := chatID.(type) {
		case lua.LNumber:
			cid = int64(v)
		case lua.LString:
			cid = string(v)
		default:
			ls.Push(lua.LNil)
			ls.Push(lua.LString("invalid chat_id"))
			return 2
		}

		_, err := e.bot.SendMessage(e.ctx, &tgbot.SendMessageParams{
			ChatID: cid,
			Text:   text,
		})
		if err != nil {
			defaultLogger.Error(ThisModule, "send_message failed: %v", err)
			ls.Push(lua.LNil)
			ls.Push(lua.LString(err.Error()))
			return 2
		}

		ls.Push(lua.LString("ok"))
		ls.Push(lua.LNil)
		return 2
	}))

	e.lu.SetField(telegramTable, "get_message", e.lu.NewFunction(func(ls *lua.LState) int {
		msg, err := e.GetMessage()
		if err != nil {
			ls.Push(lua.LNil)
			ls.Push(lua.LString(err.Error()))
			return 2
		}
		msg_tab, err := any_to_table(ls, msg)
		if err != nil {
			ls.Push(lua.LNil)
			ls.Push(lua.LString(err.Error()))
			return 2
		}
		ls.Push(msg_tab)
		ls.Push(lua.LNil)
		return 2
	}))

	e.lu.SetField(telegramTable, "call", e.lu.NewFunction(func(ls *lua.LState) int {
		number := ls.CheckString(2)
		err := e.Call(number)
		if err != nil {
			ls.Push(lua.LString(err.Error()))
			return 1
		}
		ls.Push(lua.LNil)
		return 1
	}))
	e.lu.SetField(telegramTable, "test", e.lu.NewFunction(func(ls *lua.LState) int {
		number := ls.CheckString(2)
		err := e.Test(number)
		if err != nil {
			ls.Push(lua.LString(err.Error()))
			return 1
		}
		ls.Push(lua.LNil)
		return 1
	}))
	// telegram.sleep(ms)
	e.lu.SetField(telegramTable, "sleep", e.lu.NewFunction(func(ls *lua.LState) int {
		ms := ls.CheckNumber(2)
		time.Sleep(time.Millisecond * time.Duration(ms))
		return 0
	}))

	// telegram.log(level, msg)
	e.lu.SetField(telegramTable, "log", e.lu.NewFunction(func(ls *lua.LState) int {
		level := ls.CheckString(2)
		msg := ls.CheckString(3)
		switch strings.ToLower(level) {
		case "debug":
			defaultLogger.Debug(ThisModule, "[Lua] %s", msg)
		case "info":
			defaultLogger.Info(ThisModule, "[Lua] %s", msg)
		case "warn":
			defaultLogger.Warn(ThisModule, "[Lua] %s", msg)
		case "error":
			defaultLogger.Error(ThisModule, "[Lua] %s", msg)
		default:
			defaultLogger.Info(ThisModule, "[Lua] %s", msg)
		}
		return 0
	}))

	// 执行 Lua 脚本文件
	if err := e.lu.DoFile(e.file); err != nil {

		defaultLogger.Error(ThisModule, "Lua DoFile failed: %v, script: %s", err, e.file)
		// 尝试向用户发送错误通知
		if e.update != nil && e.update.Message != nil {
			_, _ = e.bot.SendMessage(e.ctx, &tgbot.SendMessageParams{
				ChatID: e.update.Message.Chat.ID,
				Text:   fmt.Sprintf("Script execution error: %v", err),
			})
		}
	}
}

// Close 关闭 Lua 引擎
func (e *LuaEngine) Close() {
	if !e.isOk {
		return
	}
	e.isOk = false
	<-e.exit
}
