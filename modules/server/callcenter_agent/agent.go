package callcenter_agent

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upGrader websocket.Upgrader

func wsHandler(c *gin.Context) {
	// 升级 HTTP 为 WebSocket 连接
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: err.Error()})
		return
	}
	agent := newAgent(ws)
	_ = agent.run()
}

const (
	AGENT_STATE_OFFLINE = 0
	AGENT_STATE_IDLE    = 1
	AGENT_STATE_BUSY    = 2
)

const (
	EXT_STATE_OFFLINE  = 0
	EXT_STATE_IDLE     = 1
	EXT_STATE_PREEMPT  = 2
	EXT_STATE_RING     = 3
	EXT_STATE_RINGBACK = 4
	EXT_STATE_INCALL   = 5
)

const (
	AGENT_SHOW_STATUS_OFFLINE  = "OFFLINE"
	AGENT_SHOW_STATUS_IDLE     = "IDLE"
	AGENT_SHOW_STATUS_PREEMPT  = "PREEMPT"
	AGENT_SHOW_STATUS_RING     = "RING"
	AGENT_SHOW_STATUS_RINGBACK = "RINGBACK"
	AGENT_SHOW_STATUS_ONCALL   = "INCALL"
	AGENT_SHOW_STATUS_TIDY     = "TIDY"
	AGENT_SHOW_STATUS_BUSY     = "BUSY"
)

const (
	NOTIFY_CODE_LOGIN_SUCCESS = 0
	NOTIFY_CODE_LOGIN_FAILED  = 1
	NOTIFY_CODE_LOGOUT        = 2
	NOTIFY_CODE_ERROR_PARAM   = 3
	NOTIFY_MAKECALL_FAILED    = 4
	NOTIFY_AGENT_STATUS       = 100
)

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type LoginMessage struct {
	TenantId string `json:"tenant_id"`
	AgentId  string `json:"agent_id"`
	Number   string `json:"number"`
}

type MakecallMessage struct {
	DisplayNumber string `json:"display_number"` // 外显号码
	DestType      string `json:"dest_type"`      // 表示呼叫用户，坐席    user, agent
	DestNumber    string `json:"dest_number"`    // 用户号码，或者坐席工号
}

type ChangeStatusMessage struct {
	State string `json:"state"`
}

type NotifyMessage struct {
	Code int `json:"code"`
	Msg  any `json:"msg"`
}
type Agent struct {
	ws         *websocket.Conn
	agentId    string
	tenantId   string
	agentState int
	extState   int
	number     string
	db_info    store.Agent
	ev_chan    chan any
	task       *datacenter.Task
	func_chan  chan func()
	session_id string
}

func (p *Agent) notify(code int, msg any) error {
	rsp := NotifyMessage{
		Code: code,
		Msg:  msg,
	}
	defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, number: %v Notify: %v", p.tenantId, p.agentId, p.number, rsp)
	buff, _ := json.Marshal(rsp)
	return p.ws.WriteMessage(websocket.TextMessage, buff)
}

// 必须用于  run 中协程出来的方法中
func (p *Agent) async_notify(code int, msg any) {
	defer func() {
		recover()
	}()
	p.func_chan <- func() {
		_ = p.notify(code, msg)
	}
}
func (p *Agent) update_agent_status() {
	switch p.extState {
	case EXT_STATE_PREEMPT:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_PREEMPT)
		return
	case EXT_STATE_RING:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_RING)
		return
	case EXT_STATE_RINGBACK:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_RINGBACK)
		return
	case EXT_STATE_INCALL:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_ONCALL)
		return
	default:
		break
	}

	switch p.agentState {
	case AGENT_STATE_BUSY:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_BUSY)
	case AGENT_STATE_IDLE:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_IDLE)
	case AGENT_STATE_OFFLINE:
		_ = p.notify(NOTIFY_AGENT_STATUS, AGENT_SHOW_STATUS_OFFLINE)
	default:
		break
	}
}

func (p *Agent) update_agent_state(state int) {
	p.agentState = state
	p.update_agent_status()
}

func (p *Agent) update_ext_state(state int) {
	switch p.extState {
	case EXT_STATE_IDLE:
		switch state {
		case EXT_STATE_PREEMPT:
			fallthrough
		case EXT_STATE_RING:
			fallthrough
		case EXT_STATE_RINGBACK:
			fallthrough
		case EXT_STATE_INCALL:
			p.extState = state
		}
	case EXT_STATE_PREEMPT:
		switch state {
		case EXT_STATE_IDLE:
			fallthrough
		case EXT_STATE_RING:
			fallthrough
		case EXT_STATE_RINGBACK:
			fallthrough
		case EXT_STATE_INCALL:
			p.extState = state
		}
	case EXT_STATE_RING:
		switch state {
		case EXT_STATE_IDLE:
			fallthrough
		case EXT_STATE_RINGBACK:
			fallthrough
		case EXT_STATE_INCALL:
			p.extState = state
		}
	case EXT_STATE_RINGBACK:
		switch state {
		case EXT_STATE_IDLE:
			fallthrough
		case EXT_STATE_INCALL:
			p.extState = state
		}
	case EXT_STATE_INCALL:
		switch state {
		case EXT_STATE_IDLE:
			p.extState = state
		}
	}
	p.update_agent_status()
}

func (p *Agent) login(msg LoginMessage) bool {
	if msg.TenantId == "" || msg.AgentId == "" || msg.Number == "" {
		_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "参数错误")
		return false
	}
	p.agentId = msg.AgentId
	p.tenantId = msg.TenantId
	p.number = msg.Number

	// 1. 检查坐席是否签入
	agent := agent_manager.GetById(msg.TenantId, msg.AgentId)
	defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, number: %v Login", msg.TenantId, msg.AgentId, msg.Number)
	if agent != nil {
		_ = p.notify(NOTIFY_CODE_LOGIN_FAILED, "坐席已登录")
		defaultLogger.Error(ThisModule, "tenantId: %v, agentId: %v, number: %v Login Failed, agent already login", msg.TenantId, msg.AgentId, msg.Number)
		return false
	}

	// 检查坐席是库中的坐席
	info, err := app.GetDefaultApp().GetStoreEngine().QueryAgent(store.Agent{TenantId: msg.TenantId, AgentId: msg.AgentId})
	if err != nil {
		_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "坐席不存在")
		defaultLogger.Error(ThisModule, "tenantId: %v, agentId: %v, number: %v Login Failed, agent not found", msg.TenantId, msg.AgentId, msg.Number)
		return false
	}
	p.db_info = info
	// 2. 避免并发签入 ()
	if !agent_manager.CheckAndAdd(p) {
		_ = p.notify(NOTIFY_CODE_LOGIN_FAILED, "坐席已被签入")
		defaultLogger.Error(ThisModule, "tenantId: %v, agentId: %v, number: %v Login Failed, agent already logined", msg.TenantId, msg.AgentId, msg.Number)
		return false
	}

	// 3. 检查分机是否签入，如果未签入，一秒检查一次，检查 3 次
	ext := store.ExtensionManagerInstance.GetByNumber(msg.Number)
	for {
		if ext != nil {
			break
		}
		time.Sleep(time.Second)
		ext = store.ExtensionManagerInstance.GetByNumber(msg.Number)
	}
	if ext == nil {
		_ = p.notify(NOTIFY_CODE_LOGIN_FAILED, "分机未签入")
		defaultLogger.Error(ThisModule, "tenantId: %v, agentId: %v, number: %v Login Failed, extension not found", msg.TenantId, msg.AgentId, msg.Number)
		return false
	}
	p.agentState = AGENT_STATE_IDLE
	p.extState = EXT_STATE_IDLE
	p.update_agent_status()
	_ = p.notify(NOTIFY_CODE_LOGIN_SUCCESS, "成功")
	return true
}

func (p *Agent) logout() {
	defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, number: %v Logout", p.tenantId, p.agentId, p.number)
	agent_manager.Del(p)
	_ = p.ws.Close()
	close(p.ev_chan)
	close(p.func_chan)
}

func (p *Agent) makecall(msg MakecallMessage) {
	if msg.DisplayNumber == "" || msg.DestNumber == "" {
		p.async_notify(NOTIFY_CODE_ERROR_PARAM, "Number 参数为空")
		return
	}
	if msg.DestType == "" {
		msg.DestType = "user"
	}

	switch msg.DestType {
	case "user":
	case "agent":
	default:
		p.async_notify(NOTIFY_CODE_ERROR_PARAM, "DestType 参数错误")
		return
	}
	// 1. 检查外显是否正确
	defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, displayNumber: %v, destNumber:%v Makecall", p.tenantId, p.agentId, msg.DisplayNumber, msg.DestNumber)
	tenant_number, err := app.GetDefaultApp().GetStoreEngine().QueryTenantNumber(store.TenantNumber{TenantId: p.tenantId, Number: msg.DisplayNumber})
	if err != nil {
		p.async_notify(NOTIFY_CODE_ERROR_PARAM, "外显号码不存在")
		defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, displayNumber: %v, destNumber:%v Makecall Failed, tenant_number not found", p.tenantId, p.agentId, msg.DisplayNumber, msg.DestNumber)
		return
	}
	// 2. 需要检查当前坐席分机是否可以呼出
	ext := store.ExtensionManagerInstance.GetByNumber(p.number)
	if ext == nil {
		p.async_notify(NOTIFY_MAKECALL_FAILED, "分机不在线")
		defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, displayNumber: %v, destNumber:%v Makecall Failed, extension not found", p.tenantId, p.agentId, msg.DisplayNumber, msg.DestNumber)
		return
	}
	if p.extState != EXT_STATE_IDLE {
		p.async_notify(NOTIFY_MAKECALL_FAILED, "分机非空闲")
		defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, displayNumber: %v, destNumber:%v Makecall Failed, extension not idle", p.tenantId, p.agentId, msg.DisplayNumber, msg.DestNumber)
		return
	}

	// 3. 调用脚本开始呼叫
	job_uuid := uuid.New().String()
	session_A := uuid.New().String()
	session_B := uuid.New().String()
	task := datacenter.GetTaskManager().CreateAndGet(session_A)
	task.Lock()
	defer task.Unlock()
	task.TenantId = p.tenantId
	task.A_Session.TaskID = session_A
	task.A_Session.SessionID = session_A
	task.A_Session.Caller = msg.DisplayNumber
	task.A_Session.Callee = p.agentId
	task.B_Session.OriCaller = p.agentId
	task.B_Session.OriCallee = msg.DestNumber

	task.B_Session.TaskID = session_A
	task.B_Session.SessionID = session_B
	task.B_Session.Caller = msg.DisplayNumber
	task.B_Session.Callee = msg.DestNumber
	task.B_Session.OriCaller = p.agentId
	task.B_Session.OriCallee = msg.DestNumber

	datacenter.GetSessionManager().Set(task.A_Session.SessionID, task.A_Session)
	datacenter.GetSessionManager().Set(task.B_Session.SessionID, task.B_Session)
	task.TaskType = datacenter.TASK_TYPE_makecall
	agent_manager.AddSession(session_A, p)
	p.func_chan <- func() {
		p.task = task
		p.session_id = session_A
		p.update_ext_state(EXT_STATE_PREEMPT)
	}
	r := pbx.Makecall(pbx.MakecallArg{
		JobUUID:     job_uuid,
		TaskID:      task.TaskID,
		A_SessionID: task.A_Session.SessionID,
		A_Caller:    task.A_Session.Caller,
		A_Callee:    p.number,
		A_WayType:   store.WAY_TYPE_local,
		A_Way:       "",
		B_SessionID: task.B_Session.SessionID,
		B_Caller:    task.B_Session.Caller,
		B_Callee:    task.B_Session.Callee,
		B_WayType:   tenant_number.WayType,
		B_Way:       tenant_number.Way,
		Timeout:     60,
	})

	p.func_chan <- func() {
		if r.Code != 0 {
			p.update_ext_state(EXT_STATE_IDLE)
			agent_manager.DelSession(session_A)
			p.task = nil
		}
	}

	defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, displayNumber: %v, destNumber:%v Makecall Completed, code: %v, msg: %v",
		p.tenantId, p.agentId, msg.DisplayNumber, msg.DestNumber, r.Code, r.Data)
}

func (p *Agent) change_status(msg ChangeStatusMessage) {
	defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v, status: %v ChangeStatus", p.tenantId, p.agentId, msg.State)
	switch msg.State {
	case "busy":
		p.update_agent_state(AGENT_STATE_BUSY)
	case "idle":
		p.update_agent_state(AGENT_STATE_IDLE)
	}
}

func (p *Agent) hangup() {
	if p.task != nil {
		defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v Hangup Sesssion:%v",
			p.tenantId, p.agentId, p.session_id)
		pbx.HangupCall(p.session_id)
		switch p.task.TaskType {
		case datacenter.TASK_TYPE_makecall:
			if p.task.B_Session.CallTime > 0 {
				defaultLogger.Info(ThisModule, "tenantId: %v, agentId: %v Hangup Sesssion:%v",
					p.tenantId, p.agentId, p.task.B_Session.SessionID)
				pbx.HangupCall(p.task.B_Session.SessionID)
			}
		}
	}
}

func (p *Agent) PushEvent(ev any) {
	defer func() {
		recover()
	}()
	p.ev_chan <- ev
}

func (p *Agent) run() error {
	is_login := false
	ws_msg := make(chan []byte, 100)
	go func() {
		defer func() {
			recover()
		}()
		for {
			_, data, err := p.ws.ReadMessage()
			if err != nil {
				break
			}
			ws_msg <- data
		}
		ws_msg <- nil
	}()
	exit := false
	for !exit {
		select {
		case recv_event := <-p.ev_chan:
			switch ev := recv_event.(type) {
			case *event.CustomCallStatus:
				switch ev.Status {
				case "RING":
					p.update_ext_state(EXT_STATE_RING)
				case "RINGBACK":
					p.update_ext_state(EXT_STATE_RINGBACK)
				case "INCALL":
					p.update_ext_state(EXT_STATE_INCALL)
				}
			case *event.SessionHangup:
				p.update_ext_state(EXT_STATE_IDLE)
				p.task = nil
			}
		case data := <-ws_msg:
			if data == nil {
				exit = true
				break
			}
			var msg Message
			err := json.Unmarshal(data, &msg)
			if err != nil {
				_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "参数错误")
				exit = true
				break
			}
			if msg.Type != "login" && !is_login {
				_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "未登录")
				exit = true
				break
			}
			switch msg.Type {
			case "login":
				var loginMsg LoginMessage
				if err := json.Unmarshal(msg.Data, &loginMsg); err != nil {
					_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "登录参数错误")
					exit = true
					break
				}
				if !p.login(loginMsg) {
					exit = true
					break
				}
				is_login = true
			// 可扩展其他消息类型
			case "logout":
				exit = true
			case "makecall":
				var body_msg MakecallMessage
				if err := json.Unmarshal(msg.Data, &body_msg); err != nil {
					_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "参数错误")
					break
				}
				go p.makecall(body_msg)
			case "hangup":
				p.hangup()
			case "change_status":
				var body_msg ChangeStatusMessage
				if err := json.Unmarshal(msg.Data, &body_msg); err != nil {
					_ = p.notify(NOTIFY_CODE_ERROR_PARAM, "参数错误")
					break
				}
				p.change_status(body_msg)
			}
		case fun := <-p.func_chan:
			fun()
		}
	}
	close(ws_msg)
	p.logout()
	return nil
}

func newAgent(ws *websocket.Conn) *Agent {
	return &Agent{
		ws:         ws,
		agentState: AGENT_STATE_OFFLINE,
		extState:   EXT_STATE_OFFLINE,
		ev_chan:    make(chan any, 100),
		func_chan:  make(chan func(), 100),
	}
}

type AgentManager struct {
	agents map[string]*Agent
	exts   map[string]*Agent
	mu     sync.RWMutex

	agent_session_map map[string]*Agent
	agent_session_mu  sync.RWMutex
}

func (p *AgentManager) AddSession(id string, agent *Agent) {
	p.agent_session_mu.Lock()
	defer p.agent_session_mu.Unlock()
	p.agent_session_map[id] = agent
}

func (p *AgentManager) DelSession(id string) {
	p.agent_session_mu.Lock()
	defer p.agent_session_mu.Unlock()
	delete(p.agent_session_map, id)
}

func (p *AgentManager) GetAgentBySession(id string) *Agent {
	p.agent_session_mu.RLock()
	defer p.agent_session_mu.RUnlock()
	return p.agent_session_map[id]
}

func (p *AgentManager) Add(agent *Agent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.agents[agent.tenantId+"-"+agent.agentId] = agent
	p.exts[agent.number] = agent
}

func (p *AgentManager) CheckAndAdd(agent *Agent) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.agents[agent.tenantId+"-"+agent.agentId]; ok {
		return false
	}
	p.agents[agent.tenantId+"-"+agent.agentId] = agent
	p.exts[agent.number] = agent
	return true
}
func (p *AgentManager) GetById(tenant_id, agent_id string) *Agent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.agents[tenant_id+"-"+agent_id]
}

func (p *AgentManager) GetByNumber(number string) *Agent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exts[number]
}

func (p *AgentManager) Del(agent *Agent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.agents, agent.tenantId+"-"+agent.agentId)
	delete(p.exts, agent.number)
}

var agent_manager *AgentManager = &AgentManager{
	agents:            make(map[string]*Agent),
	exts:              make(map[string]*Agent),
	agent_session_map: make(map[string]*Agent),
}
