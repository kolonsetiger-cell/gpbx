package datacenter

import (
	"strings"
	"sync"
	"time"
)

const (
	Action_Create   = 1
	Action_Ring     = 2
	Action_Ringback = 3
	Action_Answer   = 4
	Action_Hangup   = 5
	Action_Destroy  = 6
	Action_Bridge   = 7
	Action_Transfer = 8
	Action_Playback = 9
	Action_DTMF     = 10
)

// Session 有效期常量 (毫秒)
const (
	SessionValidCallThreshold = 60 * 1000       // 1分钟: 需要大于1分钟才能判定是否有效的 call
	SessionDestroyThreshold   = 2 * 60 * 1000   // 2分钟: Destroy后超过此时间认为会话完成
	SessionMaxCacheDuration   = 120 * 60 * 1000 // 120分钟: 最多缓存120分钟
)

type Action struct {
	Type   int    // 动作类型
	Time   int64  // 动作时间
	Target string // 动作对象
}

type Session struct {
	TaskID       string
	SessionID    string
	Caller       string
	Callee       string
	OriCaller    string
	OriCallee    string
	CallTime     int64
	RingTime     int64
	RingbackTime int64
	AnswerTime   int64
	BridgeTime   int64
	HangupTime   int64
	DestroyTime  int64
	IsRobot      bool
	RobotID      string
	Direction    string
	createTime   int64
	dtmf_que     []string
	dtmf_mu      sync.Mutex
	dtmf_enable  bool
}

func (s *Session) PushDtmf(dtmf string) {
	s.dtmf_mu.Lock()
	defer s.dtmf_mu.Unlock()
	if s.dtmf_enable {
		s.dtmf_que = append(s.dtmf_que, dtmf)
	}
}

func (s *Session) EnableDtmf(enable bool) {
	s.dtmf_mu.Lock()
	defer s.dtmf_mu.Unlock()
	s.dtmf_enable = enable
	s.dtmf_que = s.dtmf_que[:0]
}

func (s *Session) GetDtmfs() string {
	var ret string
	s.dtmf_mu.Lock()
	defer s.dtmf_mu.Unlock()
	ret = strings.Join(s.dtmf_que, "")
	s.dtmf_que = s.dtmf_que[:0]
	return ret
}

func (s *Session) isValidCall() bool {
	now := time.Now().UnixMilli()
	if now-s.createTime < SessionValidCallThreshold {
		return true
	}
	return s.CallTime > 0 && !s.isComplete()
}

func (s *Session) isComplete() bool {
	now := time.Now().UnixMilli()
	if s.DestroyTime > 0 && now-s.DestroyTime > SessionDestroyThreshold {
		return true
	}
	if now-s.createTime > SessionMaxCacheDuration {
		return true
	}
	return false
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func (t *SessionManager) CreateAndGet(id string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	session, ok := t.sessions[id]
	if !ok {
		session = &Session{
			SessionID:  id,
			createTime: time.Now().UnixMilli(),
		}
		t.sessions[id] = session
	}
	return session
}

func (t *SessionManager) Set(id string, session *Session) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessions[id] = session
}

func (t *SessionManager) Get(id string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	session, ok := t.sessions[id]
	if !ok {
		return nil
	}
	return session
}

func (t *SessionManager) Delete(id string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	session, ok := t.sessions[id]
	if !ok {
		return nil
	}
	delete(t.sessions, id)
	return session
}

func (t *SessionManager) RemoveCompletedCall() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.sessions {
		if !v.isValidCall() {
			delete(t.sessions, k)
		}
	}
}

var sessionManager *SessionManager
var globalSessionManager *SessionManager

func GetSessionManager() *SessionManager {
	return sessionManager
}

func GetGlobalSessionManager() *SessionManager {
	return globalSessionManager
}

func init() {
	sessionManager = &SessionManager{
		sessions: make(map[string]*Session),
	}
	globalSessionManager = &SessionManager{
		sessions: make(map[string]*Session),
	}
}
