package ivr

import (
	"encoding/json"
	"path/filepath"
	"sync"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
)

type Engine interface {
	do()
	Close()
	pushDtmf(string)
}

type ivr_body struct {
	task_id    string
	session_id string
	ivr_id     string
	engine     Engine
}

func (b *ivr_body) on_answer() {
	if b.ivr_id == "test" {
		b.engine = NewLuaEngine(filepath.Join("./ivrs", "test.lua"), b.session_id)
		b.engine.do()
	} else {
		ivr, err := app.GetDefaultApp().GetStoreEngine().QueryIvr(store.Ivr{
			IvrID: b.ivr_id,
		})
		if err != nil || ivr.IvrID == "" {
			pbx.HangupCall(b.session_id)
			defaultLogger.Error(ThisModule, "Ivr id %v not found", b.ivr_id)
			return
		}
		defaultLogger.Info(ThisModule, "Session:%v Answer", b.session_id)
		switch ivr.Type {
		case store.IVR_TYPE_Dify:
			var conf DifyConf
			err = json.Unmarshal([]byte(ivr.Args), &conf)
			if err != nil {
				pbx.HangupCall(b.session_id)
				defaultLogger.Error(ThisModule, "Ivr id %v Args Invalid %v", b.ivr_id, ivr.Args)
				return
			}
			conf.Url = ivr.Path
			b.engine = NewDifyEngine(conf, b.session_id, b.task_id)
			b.engine.do()
		case store.IVR_TYPE_Lua:
			// 执行机器人LUA脚本
			b.engine = NewLuaEngine(filepath.Join("./ivrs", ivr.Path), b.session_id)
			b.engine.do()
		default:
			pbx.HangupCall(b.session_id)
			defaultLogger.Error(ThisModule, "Ivr id %v Not Support Type %v", b.ivr_id, ivr.Type)
		}
	}
}

func (b *ivr_body) on_hangup() {
	defaultLogger.Info(ThisModule, "Session:%v Hangup", b.session_id)
	if b.engine != nil {
		b.engine.Close()
	}
}

func (b *ivr_body) on_dtmf(dtmf string) {
	defaultLogger.Info(ThisModule, "Session:%v DTMF %v", b.session_id, dtmf)
	if b.engine != nil {
		b.engine.pushDtmf(dtmf)
	}
}
func newIVRBody() *ivr_body {
	return &ivr_body{}
}

type manager struct {
	mu       sync.Mutex
	body_map map[string]*ivr_body
}

func (n *manager) Get(task_id string) *ivr_body {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.body_map[task_id]
}

func (n *manager) Set(task_id string, body *ivr_body) {
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
		body_map: make(map[string]*ivr_body),
	}
}

var defaultManager = newManager()
