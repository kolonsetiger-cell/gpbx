package ai

import (
	"path/filepath"
	"sync"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
)

type ai_body struct {
	task_id    string
	session_id string
	robot_id   string
	robot      store.Robot
	lua_engine *LuaEngine
	ai_vendor  ai_vendor
}

func (b *ai_body) on_answer() {
	defaultLogger.Info(ThisModule, "Session:%v Answer", b.session_id)
	// 开始播放欢迎语音
	var err error
	b.robot, err = app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{RobotID: b.robot_id})
	if err != nil {
		defaultLogger.Error(ThisModule, "<%v> Session:%v Not Found RobotId %v", b.task_id, b.session_id, b.robot_id)
		// 未找到机器人配置 直接挂断通话
		pbx.HangupCall(b.session_id)
		return
	}
	switch app.GetDefaultApp().GetCfg().Child("llm.type").GetString() {
	case "openai":
		b.ai_vendor = NewAIOpenAI()
	case "dify":
		b.ai_vendor = NewAIDify()
	default:
		defaultLogger.Error(ThisModule, "<%v> Session:%v Not Found LLM Type %v", b.task_id, b.session_id, app.GetDefaultApp().GetCfg().Child("llm.type").GetString())
		pbx.HangupCall(b.session_id)
		return
	}

	_ = b.ai_vendor.Load(app.GetDefaultApp().GetCfg().Child("llm.model").GetString(),
		app.GetDefaultApp().GetCfg().Child("llm.baseurl").GetString(),
		app.GetDefaultApp().GetCfg().Child("llm.token").GetString(),
		int(app.GetDefaultApp().GetCfg().Child("llm.max_history").GetInt()))
	// 执行机器人LUA脚本
	b.lua_engine = NewLuaEngine(filepath.Join("./robots", b.robot_id+".lua"), b.session_id)
	b.lua_engine.setAiVendor(b.ai_vendor)
	b.lua_engine.do()
}

func (b *ai_body) on_asr(asr string) {
	defaultLogger.Info(ThisModule, "Session:%v ASR:%v", b.session_id, asr)
	if b.lua_engine == nil {
		return
	}
	if b.lua_engine.IsOk() {
		b.lua_engine.setAsr(asr)
	}
}

func (b *ai_body) on_hangup() {
	defaultLogger.Info(ThisModule, "Session:%v Hangup", b.session_id)
	if b.lua_engine != nil {
		b.lua_engine.Close()
	}
}

func newAIBody() *ai_body {
	return &ai_body{}
}

type manager struct {
	mu       sync.Mutex
	body_map map[string]*ai_body
}

func (n *manager) Get(task_id string) *ai_body {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.body_map[task_id]
}

func (n *manager) Set(task_id string, body *ai_body) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.body_map[task_id] = body
}

func (n *manager) Delete(task_id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.body_map, task_id)
}

func newManager() *manager {
	return &manager{
		body_map: make(map[string]*ai_body),
	}
}

var defaultManager = newManager()
