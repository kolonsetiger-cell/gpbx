package store

import (
	"errors"
	"sync"

	"gitee.com/kolonse_zhjsh/gpbx/log"
)

type memoryStore struct {
	// filePath string
	logger           log.Logger
	mu_user          sync.RWMutex
	users            map[string]*User
	mu_tenant        sync.RWMutex
	tenants          map[string]*Tenant
	mu_tenant_number sync.RWMutex
	numbers          map[string]*TenantNumber
	mu_robot         sync.RWMutex
	robots           map[string]*Robot
	mu_extension     sync.RWMutex
	extensions       map[string]*Extension
	mu_agent         sync.RWMutex
	agents           map[string]*Agent
	mu_ivr           sync.RWMutex
	ivrs             map[string]*Ivr
	mu_tg_user       sync.RWMutex
	tgUsers          map[string]*TeleGramUser
}

func (db *memoryStore) GetMemDB() Store {
	return db
}
func (db *memoryStore) SetLogger(logger log.Logger) {
	db.logger = logger
}

func (db *memoryStore) Load(path string) error {
	db.users = make(map[string]*User)
	db.tenants = make(map[string]*Tenant)
	db.numbers = make(map[string]*TenantNumber)
	db.robots = make(map[string]*Robot)
	db.extensions = make(map[string]*Extension)
	db.agents = make(map[string]*Agent)
	db.ivrs = make(map[string]*Ivr)
	db.tgUsers = make(map[string]*TeleGramUser)
	return nil
}

func (db *memoryStore) CreateUser(User) error {
	return nil
}

func (db *memoryStore) StoreUser(u User) error {
	db.mu_user.Lock()
	defer db.mu_user.Unlock()
	v, ok := db.users[u.Username]
	if !ok {
		db.users[u.Username] = &u
	} else {
		if len(u.Password) != 0 {
			v.Password = u.Password
		}
		if len(u.Name) != 0 {
			v.Name = u.Name
		}
		if len(u.Roles) != 0 {
			v.Roles = u.Roles
		}
	}
	return nil
}

func (db *memoryStore) QueryUser(u User) (User, error) {
	db.mu_user.RLock()
	defer db.mu_user.RUnlock()
	if v, ok := db.users[u.Username]; ok {
		return *v, nil
	}
	return User{}, errors.New("Not Found")
}

func (db *memoryStore) CountUser() int64 {
	return int64(len(db.users))
}

func (db *memoryStore) CreateTenant(Tenant) error {
	return nil
}

func (db *memoryStore) StoreTenant(t Tenant) error {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()

	v, ok := db.tenants[t.TenantId]
	if !ok {
		db.tenants[t.TenantId] = &t
	} else {
		if t.CreateTime > 0 {
			v.CreateTime = t.CreateTime
		}

		if t.ExpireTime > 0 {
			v.ExpireTime = t.ExpireTime
		}

		if len(t.DefaultNumber) != 0 {
			v.DefaultNumber = t.DefaultNumber
		}

		if len(t.TenantName) != 0 {
			v.TenantName = t.TenantName
		}
	}
	return nil
}

func (db *memoryStore) QueryTenant(t Tenant) (Tenant, error) {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()

	if v, ok := db.tenants[t.TenantId]; ok {
		return *v, nil
	}
	return Tenant{}, errors.New("Not Found")
}

func (db *memoryStore) CountTenant() int64 {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()
	return int64(len(db.tenants))
}

func (db *memoryStore) QueryTenantOfPage(offset int64, size int64) ([]Tenant, error) {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()
	var records []Tenant
	for _, v := range db.tenants {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteTenant(t Tenant) error {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()
	delete(db.tenants, t.TenantId)
	return nil
}

func (db *memoryStore) CreateTenantNumber(TenantNumber) error {
	return nil
}

func (db *memoryStore) StoreTenantNumber(t TenantNumber) error {
	db.mu_tenant_number.Lock()
	defer db.mu_tenant_number.Unlock()

	v, ok := db.numbers[t.Number]
	if !ok {
		db.numbers[t.Number] = &t
	} else {
		if t.CreateTime > 0 {
			v.CreateTime = t.CreateTime
		}

		if t.ExpireTime > 0 {
			v.ExpireTime = t.ExpireTime
		}

		if t.Action != 0 {
			v.Action = t.Action
		}

		if len(t.WayType) != 0 {
			v.WayType = t.WayType
		}
		if len(t.Way) != 0 {
			v.Way = t.Way
		}
	}
	return nil
}

func (db *memoryStore) QueryTenantNumber(t TenantNumber) (TenantNumber, error) {
	db.mu_tenant_number.Lock()
	defer db.mu_tenant_number.Unlock()
	if v, ok := db.numbers[t.Number]; ok {
		return *v, nil
	}
	return TenantNumber{}, errors.New("Not Found")
}

// QueryTenantNumbers 查询租户号码列表，支持多条件组合查询（AND关系）
// 查询条件说明：
//   - Number: 非空字符串作为查询条件
//   - TenantId: 非空字符串作为查询条件
//   - Action: 非0值作为查询条件
//   - WayType: 非空字符串作为查询条件
//   - Way: 非空字符串作为查询条件
//   - RobotID: 非空字符串作为查询条件
//   - CreateTime/ExpireTime: 不作为查询条件，始终匹配所有记录
func (db *memoryStore) QueryTenantNumbers(t TenantNumber) ([]TenantNumber, error) {
	db.mu_tenant_number.RLock()
	defer db.mu_tenant_number.RUnlock()
	var results []TenantNumber
	for _, v := range db.numbers {
		// Number: 非空字符串作为查询条件
		if t.Number != "" && v.Number != t.Number {
			continue
		}
		// TenantId: 非空字符串作为查询条件
		if t.TenantId != "" && v.TenantId != t.TenantId {
			continue
		}
		// Action: 非0作为查询条件
		if t.Action != 0 && v.Action != t.Action {
			continue
		}
		// WayType: 非空字符串作为查询条件
		if t.WayType != "" && v.WayType != t.WayType {
			continue
		}
		// Way: 非空字符串作为查询条件
		if t.Way != "" && v.Way != t.Way {
			continue
		}
		// RobotID: 非空字符串作为查询条件
		if t.RobotID != "" && v.RobotID != t.RobotID {
			continue
		}
		results = append(results, *v)
	}
	return results, nil
}

func (db *memoryStore) CountTenantNumber() int64 {
	db.mu_tenant_number.Lock()
	defer db.mu_tenant_number.Unlock()
	return int64(len(db.numbers))
}

func (db *memoryStore) QueryTenantNumberOfPage(offset int64, size int64) ([]TenantNumber, error) {
	db.mu_tenant_number.Lock()
	defer db.mu_tenant_number.Unlock()
	var records []TenantNumber
	for _, v := range db.numbers {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteTenantNumber(t TenantNumber) error {
	db.mu_tenant_number.Lock()
	defer db.mu_tenant_number.Unlock()
	delete(db.numbers, t.Number)
	return nil
}

func (db *memoryStore) CreateRobot(Robot) error {
	return nil
}

func (db *memoryStore) StoreRobot(t Robot) error {
	db.mu_robot.Lock()
	defer db.mu_robot.Unlock()

	v, ok := db.robots[t.RobotID]
	if !ok {
		db.robots[t.RobotID] = &t
	} else {
		if len(t.Target) != 0 {
			v.Target = t.Target
		}
		if len(t.Arg) != 0 {
			v.Arg = t.Arg
		}
		if len(t.Prompt) != 0 {
			v.Prompt = t.Prompt
		}
		if len(t.Welcome) != 0 {
			v.Welcome = t.Welcome
		}
		if t.ToVendor != v.ToVendor {
			v.ToVendor = t.ToVendor
		}
		if t.CreateTime > 0 {
			v.CreateTime = t.CreateTime
		}
	}
	return nil
}

func (db *memoryStore) QueryRobot(t Robot) (Robot, error) {
	db.mu_robot.Lock()
	defer db.mu_robot.Unlock()
	if v, ok := db.robots[t.RobotID]; ok {
		return *v, nil
	}
	return Robot{}, errors.New("Not Found")
}

func (db *memoryStore) CountRobot() int64 {
	db.mu_robot.RLock()
	defer db.mu_robot.RUnlock()
	return int64(len(db.robots))
}

func (db *memoryStore) QueryRobotOfPage(offset int64, size int64) ([]Robot, error) {
	db.mu_robot.Lock()
	defer db.mu_robot.Unlock()
	var records []Robot
	for _, v := range db.robots {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteRobot(t Robot) error {
	db.mu_robot.Lock()
	defer db.mu_robot.Unlock()
	delete(db.robots, t.RobotID)
	return nil
}

func (db *memoryStore) CreateExtension(Extension) error {
	return nil
}

func (db *memoryStore) StoreExtension(e Extension) error {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()

	v, ok := db.extensions[e.ExtensionId]
	if !ok {
		db.extensions[e.ExtensionId] = &e
	} else {
		if e.TenantId != "" {
			v.TenantId = e.TenantId
		}
		if e.CreateTime > 0 {
			v.CreateTime = e.CreateTime
		}
		v.Status = e.Status
		v.NetworkIP = e.NetworkIP
		v.NetworkPort = e.NetworkPort
	}
	return nil
}

func (db *memoryStore) QueryExtension(e Extension) (Extension, error) {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()
	if v, ok := db.extensions[e.ExtensionId]; ok {
		return *v, nil
	}
	return Extension{}, errors.New("Not Found")
}

func (db *memoryStore) CountExtension() int64 {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()
	return int64(len(db.extensions))
}

func (db *memoryStore) QueryExtensionOfPage(offset int64, size int64) ([]Extension, error) {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()
	var records []Extension
	for _, v := range db.extensions {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteExtension(e Extension) error {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()
	delete(db.extensions, e.ExtensionId)
	return nil
}

func (db *memoryStore) CreateAgent(Agent) error {
	return nil
}

func (db *memoryStore) StoreAgent(a Agent) error {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()

	v, ok := db.agents[a.AgentId]
	if !ok {
		db.agents[a.AgentId] = &a
	} else {
		if a.TenantId != "" {
			v.TenantId = a.TenantId
		}
		if a.AgentName != "" {
			v.AgentName = a.AgentName
		}
		if a.ExtensionId != "" {
			v.ExtensionId = a.ExtensionId
		}
		if a.DisplayNumber != "" {
			v.DisplayNumber = a.DisplayNumber
		}
		if a.CreateTime > 0 {
			v.CreateTime = a.CreateTime
		}
	}
	return nil
}

func (db *memoryStore) QueryAgent(a Agent) (Agent, error) {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()
	if v, ok := db.agents[a.AgentId]; ok {
		return *v, nil
	}
	return Agent{}, errors.New("Not Found")
}

func (db *memoryStore) CountAgent() int64 {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()
	return int64(len(db.agents))
}

func (db *memoryStore) QueryAgentOfPage(offset int64, size int64) ([]Agent, error) {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()
	var records []Agent
	for _, v := range db.agents {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteAgent(a Agent) error {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()
	delete(db.agents, a.AgentId)
	return nil
}

func (db *memoryStore) CreateIvr(Ivr) error {
	return nil
}

func (db *memoryStore) StoreIvr(t Ivr) error {
	db.mu_ivr.Lock()
	defer db.mu_ivr.Unlock()

	v, ok := db.ivrs[t.IvrID]
	if !ok {
		db.ivrs[t.IvrID] = &t
	} else {
		if t.Type != "" {
			v.Type = t.Type
		}
		if len(t.Path) != 0 {
			v.Path = t.Path
		}
		if len(t.Args) != 0 {
			v.Args = t.Args
		}
		if t.CreateTime > 0 {
			v.CreateTime = t.CreateTime
		}
	}
	return nil
}

func (db *memoryStore) QueryIvr(t Ivr) (Ivr, error) {
	db.mu_ivr.RLock()
	defer db.mu_ivr.RUnlock()
	if v, ok := db.ivrs[t.IvrID]; ok {
		return *v, nil
	}
	return Ivr{}, errors.New("Not Found")
}

func (db *memoryStore) CountIvr() int64 {
	db.mu_ivr.RLock()
	defer db.mu_ivr.RUnlock()
	return int64(len(db.ivrs))
}

func (db *memoryStore) QueryIvrOfPage(offset int64, size int64) ([]Ivr, error) {
	db.mu_ivr.RLock()
	defer db.mu_ivr.RUnlock()
	var records []Ivr
	for _, v := range db.ivrs {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteIvr(t Ivr) error {
	db.mu_ivr.Lock()
	defer db.mu_ivr.Unlock()
	delete(db.ivrs, t.IvrID)
	return nil
}

func (db *memoryStore) CreateTeleGramUser(TeleGramUser) error {
	return nil
}

func (db *memoryStore) StoreTeleGramUser(t TeleGramUser) error {
	db.mu_tg_user.Lock()
	defer db.mu_tg_user.Unlock()

	v, ok := db.tgUsers[t.Username]
	if !ok {
		db.tgUsers[t.Username] = &t
	} else {
		if t.TenantId != "" {
			v.TenantId = t.TenantId
		}
		if t.AuthTime > 0 {
			v.AuthTime = t.AuthTime
		}
		if t.ExpireTime > 0 {
			v.ExpireTime = t.ExpireTime
		}
		if t.BindScript != "" {
			v.BindScript = t.BindScript
		}
		if t.BindNumbers != "" {
			v.BindNumbers = t.BindNumbers
		}
		if t.IvrId != "" {
			v.IvrId = t.IvrId
		}
	}
	return nil
}

func (db *memoryStore) QueryTeleGramUser(t TeleGramUser) (TeleGramUser, error) {
	db.mu_tg_user.RLock()
	defer db.mu_tg_user.RUnlock()
	if v, ok := db.tgUsers[t.Username]; ok {
		return *v, nil
	}
	return TeleGramUser{}, errors.New("Not Found")
}

func (db *memoryStore) CountTeleGramUser() int64 {
	db.mu_tg_user.RLock()
	defer db.mu_tg_user.RUnlock()
	return int64(len(db.tgUsers))
}

func (db *memoryStore) QueryTeleGramUserOfPage(offset int64, size int64) ([]TeleGramUser, error) {
	db.mu_tg_user.RLock()
	defer db.mu_tg_user.RUnlock()
	var records []TeleGramUser
	for _, v := range db.tgUsers {
		records = append(records, *v)
	}
	if size < 0 {
		return records, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset+size > int64(len(records)) {
		size = int64(len(records)) - offset
	}
	return records[offset : offset+size], nil
}

func (db *memoryStore) DeleteTeleGramUser(t TeleGramUser) error {
	db.mu_tg_user.Lock()
	defer db.mu_tg_user.Unlock()
	delete(db.tgUsers, t.Username)
	return nil
}

func (db *memoryStore) Unload() error {
	return nil
}

func init() {
	engines["memory://"] = &memoryStore{}
}
