package store

import (
	"strings"
	"sync"
	"time"
)

// ExtensionRegisterInfo 分机注册信息
type ExtensionRegisterInfo struct {
	Number       string
	RegisterTime time.Time
	Expires      int
	CallID       string
	NetworkIP    string
	NetworkPort  string
}

// IsValid 检查注册是否仍在有效期内
func (p *ExtensionRegisterInfo) IsValid() bool {
	return time.Since(p.RegisterTime) <= time.Duration(p.Expires)*time.Second
}

// ShouldClear 检查注册是否过期需要清理（过期时间+10秒缓冲）
func (p *ExtensionRegisterInfo) ShouldClear() bool {
	return time.Since(p.RegisterTime) > time.Duration(p.Expires+10)*time.Second
}

// Refresh 刷新注册时间和过期时间
func (p *ExtensionRegisterInfo) Refresh(expires int) {
	p.RegisterTime = time.Now()
	p.Expires = expires
}

// NewExtensionRegisterInfo 创建新的分机注册信息
func NewExtensionRegisterInfo(number string, expires int, callID string) *ExtensionRegisterInfo {
	return &ExtensionRegisterInfo{
		Number:       number,
		RegisterTime: time.Now(),
		Expires:      expires,
		CallID:       callID,
	}
}

// ExtensionRegisterManager 分机注册管理器
type ExtensionRegisterManager struct {
	infosByCallID map[string]*ExtensionRegisterInfo
	infosByNumber map[string]*ExtensionRegisterInfo
	mu            sync.RWMutex
}

func (m *ExtensionRegisterManager) GetAllOnlineByTenant(tenant_id string) []*ExtensionRegisterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ret []*ExtensionRegisterInfo
	for _, v := range m.infosByNumber {
		if strings.HasPrefix(v.Number, tenant_id) {
			ret = append(ret, v)
		}
	}
	return ret
}

// Add 添加分机注册信息
func (m *ExtensionRegisterManager) Add(e *ExtensionRegisterInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infosByCallID[e.CallID] = e
	m.infosByNumber[e.Number] = e
}

// GetByCallID 通过 Call-ID 获取分机注册信息
func (m *ExtensionRegisterManager) GetByCallID(callID string) *ExtensionRegisterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infosByCallID[callID]
}

// GetByNumber 通过号码获取分机注册信息
func (m *ExtensionRegisterManager) GetByNumber(number string) *ExtensionRegisterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infosByNumber[number]
}

// DelByCallID 通过 Call-ID 删除分机注册信息
func (m *ExtensionRegisterManager) DelByCallID(callID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.infosByCallID[callID]
	if e == nil {
		return
	}
	delete(m.infosByCallID, callID)
	delete(m.infosByNumber, e.Number)
}

// DelByNumber 通过号码删除分机注册信息
func (m *ExtensionRegisterManager) DelByNumber(number string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.infosByNumber[number]
	if e == nil {
		return
	}
	delete(m.infosByCallID, e.CallID)
	delete(m.infosByNumber, number)
}

// Del 删除指定的分机注册信息
func (m *ExtensionRegisterManager) Del(e *ExtensionRegisterInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e == nil {
		return
	}
	delete(m.infosByCallID, e.CallID)
	delete(m.infosByNumber, e.Number)
}

// ClearExpire 清理过期的分机注册信息，返回已清理的分机列表供调用方处理（如更新状态为离线）
func (m *ExtensionRegisterManager) ClearExpire() (cleared []ExtensionRegisterInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.infosByCallID {
		if e.ShouldClear() {
			delete(m.infosByCallID, e.CallID)
			delete(m.infosByNumber, e.Number)
			cleared = append(cleared, *e)
		}
	}
	return
}

// ExtensionManagerInstance 全局分机管理器实例
var ExtensionManagerInstance = &ExtensionRegisterManager{
	infosByCallID: make(map[string]*ExtensionRegisterInfo),
	infosByNumber: make(map[string]*ExtensionRegisterInfo),
}
