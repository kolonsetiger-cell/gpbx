package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gitee.com/kolonse_zhjsh/gpbx/kcfg"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"gorm.io/gorm"
)

const ThisModule = "Store"

func isExist(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func mkDirs(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func get_curdir() (string, error) {
	exePath := os.Args[0]
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return "", err
	}
	// 获取所在目录
	dir := filepath.Dir(absPath)
	return dir, nil
}

type kcfgStore struct {
	filePath         string
	logger           log.Logger
	mem_engine       Store
	mu_user          sync.RWMutex
	mu_tenant        sync.RWMutex
	mu_tenant_number sync.RWMutex
	mu_robot         sync.RWMutex
	mu_extension     sync.RWMutex
	mu_agent         sync.RWMutex
	mu_ivr           sync.RWMutex
}

func (db *kcfgStore) GetMemDB() Store {
	return db.mem_engine
}
func (db *kcfgStore) SetLogger(logger log.Logger) {
	db.logger = logger
}

// load User cfg
func (db *kcfgStore) load_user(file string) {
	db.mu_user.Lock()
	defer db.mu_user.Unlock()
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	user := User{
		ID:        uint(cfg.Child("user.ID").GetInt()),
		Username:  cfg.Child("user.Username").GetString(),
		Password:  cfg.Child("user.Password").GetString(),
		Name:      cfg.Child("user.Name").GetString(),
		Roles:     cfg.Child("user.Roles").GetString(),
		CreatedAt: cfg.Child("user.CreateTime").GetInt(),
		UpdatedAt: cfg.Child("user.UpdateTime").GetInt(),
		DeletedAt: gorm.DeletedAt{},
	}
	db.logger.Info(ThisModule, "Load User %v", user)
	db.mem_engine.StoreUser(user)
}

// load Tenant cfg
func (db *kcfgStore) load_tenant(file string) {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	tenant := Tenant{
		TenantId:      cfg.Child("tenant.TenantId").GetString(),
		TenantName:    cfg.Child("tenant.TenantName").GetString(),
		CreateTime:    cfg.Child("tenant.CreateTime").GetInt(),
		ExpireTime:    cfg.Child("tenant.ExpireTime").GetInt(),
		DefaultNumber: cfg.Child("tenant.DefaultNumber").GetString(),
	}
	db.logger.Info(ThisModule, "Load Tenant %v", tenant)
	db.mem_engine.StoreTenant(tenant)
}

func (db *kcfgStore) load_tenant_number(file string) {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	tenant_number := TenantNumber{
		Number:     cfg.Child("tenant_number.Number").GetString(),
		TenantId:   cfg.Child("tenant_number.TenantId").GetString(),
		CreateTime: cfg.Child("tenant_number.CreateTime").GetInt(),
		ExpireTime: cfg.Child("tenant_number.ExpireTime").GetInt(),
		Action:     cfg.Child("tenant_number.Action").GetInt(),
		WayType:    cfg.Child("tenant_number.WayType").GetString(),
		Way:        cfg.Child("tenant_number.Way").GetString(),
		RobotID:    cfg.Child("tenant_number.RobotID").GetString(),
	}
	db.logger.Info(ThisModule, "Load TenantNumber %v", tenant_number)
	db.mem_engine.StoreTenantNumber(tenant_number)
}

func (db *kcfgStore) load_robot(file string) {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	robot := Robot{
		RobotID:    cfg.Child("robot.RobotID").GetString(),
		Target:     cfg.Child("robot.Target").GetString(),
		Arg:        make(map[string]any),
		Welcome:    cfg.Child("robot.Welcome").GetString(),
		Prompt:     cfg.Child("robot.Prompt").GetString(),
		CreateTime: cfg.Child("robot.CreateTime").GetInt(),
		ToVendor:   cfg.Child("robot.ToVendor").GetBool(),
	}
	json.Unmarshal([]byte(cfg.Child("robot.Arg").GetString()), &robot.Arg)
	db.logger.Info(ThisModule, "Load Robot %v", robot)
	db.mem_engine.StoreRobot(robot)
}

// load Extension cfg
func (db *kcfgStore) load_extension(file string) {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	extension := Extension{
		TenantId:    cfg.Child("extension.TenantId").GetString(),
		ExtensionId: cfg.Child("extension.ExtensionId").GetString(),
		CreateTime:  cfg.Child("extension.CreateTime").GetInt(),
		Status:      cfg.Child("extension.status").GetString(),
		NetworkIP:   cfg.Child("extension.networkIp").GetString(),
		NetworkPort: cfg.Child("extension.networkPort").GetString(),
	}
	db.logger.Info(ThisModule, "Load Extension %v", extension)
	db.mem_engine.StoreExtension(extension)
}

// load Agent cfg
func (db *kcfgStore) load_agent(file string) {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	agent := Agent{
		TenantId:      cfg.Child("agent.TenantId").GetString(),
		AgentId:       cfg.Child("agent.AgentId").GetString(),
		AgentName:     cfg.Child("agent.AgentName").GetString(),
		ExtensionId:   cfg.Child("agent.ExtensionId").GetString(),
		DisplayNumber: cfg.Child("agent.DisplayNumber").GetString(),
		CreateTime:    cfg.Child("agent.CreateTime").GetInt(),
	}
	db.logger.Info(ThisModule, "Load Agent %v", agent)
	db.mem_engine.StoreAgent(agent)
}

// load Ivr cfg
func (db *kcfgStore) load_ivr(file string) {
	db.mu_ivr.Lock()
	defer db.mu_ivr.Unlock()
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			db.logger.Error(ThisModule, "Load Cfg %v Failed, err:%v", file, err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	ivr := Ivr{
		IvrID:      cfg.Child("ivr.IvrID").GetString(),
		Type:       cfg.Child("ivr.Type").GetString(),
		Path:       cfg.Child("ivr.Path").GetString(),
		Args:       cfg.Child("ivr.Args").GetString(),
		CreateTime: cfg.Child("ivr.CreateTime").GetInt(),
	}
	db.logger.Info(ThisModule, "Load Ivr %v", ivr)
	db.mem_engine.StoreIvr(ivr)
}

func (db *kcfgStore) Load(path string) error {
	db.mem_engine = Get("memory://")
	db.mem_engine.SetLogger(db.logger)
	db.mem_engine.Load(path)
	pos := strings.Index(path, "file://")
	if pos != 0 {
		return errors.New("Path error")
	}

	cur_dir, _ := get_curdir()
	db.filePath = path[7:]
	db.filePath = filepath.Join(cur_dir, db.filePath)
	if !isExist(db.filePath) {
		// filepath.
		mkDirs(db.filePath)
	}
	db.CreateAgent(Agent{})
	db.CreateExtension(Extension{})
	db.CreateRobot(Robot{})
	db.CreateIvr(Ivr{})
	db.CreateTenantNumber(TenantNumber{})
	db.CreateTenant(Tenant{})
	//  遍历目录下文件，获取到所有租户
	db.logger.Debug(ThisModule, "Check Path : %v", db.filePath)
	filepath.WalkDir(db.filePath, func(path string, d os.DirEntry, err error) error {
		if err == nil {
			if d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".kcfg" {
				return nil
			}
			filename := filepath.Base(path)
			db.logger.Debug(ThisModule, "Check File : %v %v", filename, ext)
			if strings.Index(filename, "Tenant_") == 0 {
				db.load_tenant(path)
			} else if strings.Index(filename, "TenantNumber_") == 0 {
				db.load_tenant_number(path)
			} else if strings.Index(filename, "Robot_") == 0 {
				db.load_robot(path)
			} else if strings.Index(filename, "User_") == 0 {
				db.load_user(path)
			} else if strings.Index(filename, "Extension_") == 0 {
				db.load_extension(path)
			} else if strings.Index(filename, "Agent_") == 0 {
				db.load_agent(path)
			} else if strings.Index(filename, "Ivr_") == 0 {
				db.load_ivr(path)
			}
		}
		return nil
	})
	return nil
}

func (db *kcfgStore) CreateUser(User) error {
	// 确保 User 目录存在
	userDir := filepath.Join(db.filePath, "User")
	if !isExist(userDir) {
		if err := mkDirs(userDir); err != nil {
			db.logger.Error(ThisModule, "Create User Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreUser(u User) error {
	db.mu_user.Lock()
	defer db.mu_user.Unlock()
	u.HashPassword()
	cfg := kcfg.NewCfg()
	cfg.Add("user.ID", strconv.Itoa(int(u.ID)))
	cfg.Add("user.Username", u.Username)
	cfg.Add("user.Password", u.Password)
	cfg.Add("user.Name", u.Name)
	cfg.Add("user.Roles", u.Roles)
	cfg.Add("user.CreateTime", strconv.Itoa(int(u.CreatedAt)))
	cfg.Add("user.UpdateTime", strconv.Itoa(int(u.UpdatedAt)))

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "User", "User_"+u.Username+".kcfg"), []byte(content), 0644)
	return db.mem_engine.StoreUser(u)
}

func (db *kcfgStore) QueryUser(u User) (User, error) {
	return db.mem_engine.QueryUser(u)
}

func (db *kcfgStore) CountUser() int64 {
	return db.mem_engine.CountUser()
}

func (db *kcfgStore) CreateTenant(Tenant) error {
	dir := filepath.Join(db.filePath, "Tenant")
	if !isExist(dir) {
		if err := mkDirs(dir); err != nil {
			db.logger.Error(ThisModule, "Create Tenant Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreTenant(t Tenant) error {
	db.mu_tenant.Lock()
	defer db.mu_tenant.Unlock()

	db.mem_engine.StoreTenant(t)
	new_t, _ := db.mem_engine.QueryTenant(t)
	cfg := kcfg.NewCfg()
	cfg.Add("tenant.TenantId", new_t.TenantId)
	cfg.Add("tenant.TenantName", new_t.TenantName)
	cfg.Add("tenant.CreateTime", strconv.Itoa(int(new_t.CreateTime)))
	cfg.Add("tenant.ExpireTime", strconv.Itoa(int(new_t.ExpireTime)))
	cfg.Add("tenant.DefaultNumber", new_t.DefaultNumber)

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "Tenant", "Tenant_"+new_t.TenantId+".kcfg"), []byte(content), 0644)
	return nil
}

func (db *kcfgStore) QueryTenant(t Tenant) (Tenant, error) {
	return db.mem_engine.QueryTenant(t)
}

func (db *kcfgStore) CountTenant() int64 {
	return db.mem_engine.CountTenant()
}

func (db *kcfgStore) QueryTenantOfPage(offset int64, size int64) ([]Tenant, error) {
	return db.mem_engine.QueryTenantOfPage(offset, size)
}

func (db *kcfgStore) DeleteTenant(t Tenant) error {
	os.Remove(filepath.Join(db.filePath, "Tenant", "Tenant_"+t.TenantId+".kcfg"))
	return db.mem_engine.DeleteTenant(t)
}

func (db *kcfgStore) CreateTenantNumber(TenantNumber) error {
	dir := filepath.Join(db.filePath, "TenantNumber")
	if !isExist(dir) {
		if err := mkDirs(dir); err != nil {
			db.logger.Error(ThisModule, "Create Tenant Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreTenantNumber(t TenantNumber) error {
	db.mu_tenant_number.Lock()
	defer db.mu_tenant_number.Unlock()

	db.mem_engine.StoreTenantNumber(t)
	new_t, _ := db.mem_engine.QueryTenantNumber(t)
	cfg := kcfg.NewCfg()
	cfg.Add("tenant_number.Number", new_t.Number)
	cfg.Add("tenant_number.TenantId", new_t.TenantId)
	cfg.Add("tenant_number.CreateTime", strconv.Itoa(int(new_t.CreateTime)))
	cfg.Add("tenant_number.ExpireTime", strconv.Itoa(int(new_t.ExpireTime)))
	cfg.Add("tenant_number.Action", strconv.Itoa(int(new_t.Action)))
	cfg.Add("tenant_number.WayType", new_t.WayType)
	cfg.Add("tenant_number.Way", new_t.Way)
	cfg.Add("tenant_number.RobotID", new_t.RobotID)

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "TenantNumber", "TenantNumber_"+new_t.Number+".kcfg"), []byte(content), 0644)
	return nil
}

func (db *kcfgStore) QueryTenantNumber(t TenantNumber) (TenantNumber, error) {
	return db.mem_engine.QueryTenantNumber(t)
}

// QueryTenantNumbers 查询租户号码列表，支持多条件组合查询（AND关系）
// 详细说明见 memoryStore 实现
func (db *kcfgStore) QueryTenantNumbers(t TenantNumber) ([]TenantNumber, error) {
	return db.mem_engine.QueryTenantNumbers(t)
}

func (db *kcfgStore) CountTenantNumber() int64 {
	return db.mem_engine.CountTenantNumber()
}

func (db *kcfgStore) QueryTenantNumberOfPage(offset int64, size int64) ([]TenantNumber, error) {
	return db.mem_engine.QueryTenantNumberOfPage(offset, size)
}

func (db *kcfgStore) DeleteTenantNumber(t TenantNumber) error {
	os.Remove(filepath.Join(db.filePath, "TenantNumber", "TenantNumber_"+t.Number+".kcfg"))
	return db.mem_engine.DeleteTenantNumber(t)
}

func (db *kcfgStore) CreateRobot(Robot) error {
	dir := filepath.Join(db.filePath, "Robot")
	if !isExist(dir) {
		if err := mkDirs(dir); err != nil {
			db.logger.Error(ThisModule, "Create Tenant Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreRobot(t Robot) error {
	db.mu_robot.Lock()
	defer db.mu_robot.Unlock()
	arg, _ := json.MarshalIndent(t.Arg, " ", "  ")

	db.mem_engine.StoreRobot(t)
	new_t, _ := db.mem_engine.QueryRobot(t)

	cfg := kcfg.NewCfg()
	cfg.Add("robot.RobotID", new_t.RobotID)
	cfg.Add("robot.Target", new_t.Target)
	cfg.Add("robot.Arg", string(arg))
	cfg.Add("robot.Welcome", new_t.Welcome)
	cfg.Add("robot.Prompt", new_t.Prompt)
	cfg.Add("robot.CreateTime", strconv.Itoa(int(new_t.CreateTime)))
	cfg.Add("robot.ToVendor", strconv.FormatBool(new_t.ToVendor))

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "Robot", "Robot_"+new_t.RobotID+".kcfg"), []byte(content), 0644)
	return db.mem_engine.StoreRobot(t)
}

func (db *kcfgStore) QueryRobot(t Robot) (Robot, error) {
	return db.mem_engine.QueryRobot(t)
}

func (db *kcfgStore) CountRobot() int64 {
	return db.mem_engine.CountRobot()
}

func (db *kcfgStore) QueryRobotOfPage(offset int64, size int64) ([]Robot, error) {
	return db.mem_engine.QueryRobotOfPage(offset, size)
}

func (db *kcfgStore) DeleteRobot(t Robot) error {
	os.Remove(filepath.Join(db.filePath, "Robot", "Robot_"+t.RobotID+".kcfg"))
	return db.mem_engine.DeleteRobot(t)
}

func (db *kcfgStore) CreateExtension(Extension) error {
	dir := filepath.Join(db.filePath, "Extension")
	db.logger.Debug(ThisModule, "CreateExtension dir:%s", dir)
	if !isExist(dir) {
		if err := mkDirs(dir); err != nil {
			db.logger.Error(ThisModule, "Create Extension Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreExtension(e Extension) error {
	db.mu_extension.Lock()
	defer db.mu_extension.Unlock()

	db.mem_engine.StoreExtension(e)
	new_e, _ := db.mem_engine.QueryExtension(e)
	cfg := kcfg.NewCfg()
	cfg.Add("extension.TenantId", new_e.TenantId)
	cfg.Add("extension.ExtensionId", new_e.ExtensionId)
	cfg.Add("extension.CreateTime", strconv.FormatInt(new_e.CreateTime, 10))
	cfg.Add("extension.status", new_e.Status)
	cfg.Add("extension.networkIp", new_e.NetworkIP)
	cfg.Add("extension.networkPort", new_e.NetworkPort)

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "Extension", "Extension_"+new_e.ExtensionId+".kcfg"), []byte(content), 0644)
	return nil
}

func (db *kcfgStore) QueryExtension(e Extension) (Extension, error) {
	return db.mem_engine.QueryExtension(e)
}

func (db *kcfgStore) CountExtension() int64 {
	return db.mem_engine.CountExtension()
}

func (db *kcfgStore) QueryExtensionOfPage(offset int64, size int64) ([]Extension, error) {
	return db.mem_engine.QueryExtensionOfPage(offset, size)
}

func (db *kcfgStore) DeleteExtension(e Extension) error {
	os.Remove(filepath.Join(db.filePath, "Extension", "Extension_"+e.ExtensionId+".kcfg"))
	return db.mem_engine.DeleteExtension(e)
}

func (db *kcfgStore) CreateAgent(Agent) error {
	dir := filepath.Join(db.filePath, "Agent")
	if !isExist(dir) {
		if err := mkDirs(dir); err != nil {
			db.logger.Error(ThisModule, "Create Agent Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreAgent(a Agent) error {
	db.mu_agent.Lock()
	defer db.mu_agent.Unlock()

	db.mem_engine.StoreAgent(a)
	new_a, _ := db.mem_engine.QueryAgent(a)
	cfg := kcfg.NewCfg()
	cfg.Add("agent.TenantId", new_a.TenantId)
	cfg.Add("agent.AgentId", new_a.AgentId)
	cfg.Add("agent.AgentName", new_a.AgentName)
	cfg.Add("agent.ExtensionId", new_a.ExtensionId)
	cfg.Add("agent.DisplayNumber", new_a.DisplayNumber)
	cfg.Add("agent.CreateTime", strconv.FormatInt(new_a.CreateTime, 10))

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "Agent", "Agent_"+new_a.AgentId+".kcfg"), []byte(content), 0644)
	return nil
}

func (db *kcfgStore) QueryAgent(a Agent) (Agent, error) {
	return db.mem_engine.QueryAgent(a)
}

func (db *kcfgStore) CountAgent() int64 {
	return db.mem_engine.CountAgent()
}

func (db *kcfgStore) QueryAgentOfPage(offset int64, size int64) ([]Agent, error) {
	return db.mem_engine.QueryAgentOfPage(offset, size)
}

func (db *kcfgStore) DeleteAgent(a Agent) error {
	os.Remove(filepath.Join(db.filePath, "Agent", "Agent_"+a.AgentId+".kcfg"))
	return db.mem_engine.DeleteAgent(a)
}

func (db *kcfgStore) CreateIvr(Ivr) error {
	dir := filepath.Join(db.filePath, "Ivr")
	if !isExist(dir) {
		if err := mkDirs(dir); err != nil {
			db.logger.Error(ThisModule, "Create Ivr Dir Failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (db *kcfgStore) StoreIvr(t Ivr) error {
	db.mu_ivr.Lock()
	defer db.mu_ivr.Unlock()

	db.mem_engine.StoreIvr(t)
	new_t, _ := db.mem_engine.QueryIvr(t)

	cfg := kcfg.NewCfg()
	cfg.Add("ivr.IvrID", new_t.IvrID)
	cfg.Add("ivr.Type", new_t.Type)
	cfg.Add("ivr.Path", new_t.Path)
	cfg.Add("ivr.Args", new_t.Args)
	cfg.Add("ivr.CreateTime", strconv.FormatInt(new_t.CreateTime, 10))

	content := cfg.Dump()
	os.WriteFile(filepath.Join(db.filePath, "Ivr", "Ivr_"+new_t.IvrID+".kcfg"), []byte(content), 0644)
	return nil
}

func (db *kcfgStore) QueryIvr(t Ivr) (Ivr, error) {
	return db.mem_engine.QueryIvr(t)
}

func (db *kcfgStore) CountIvr() int64 {
	return db.mem_engine.CountIvr()
}

func (db *kcfgStore) QueryIvrOfPage(offset int64, size int64) ([]Ivr, error) {
	return db.mem_engine.QueryIvrOfPage(offset, size)
}

func (db *kcfgStore) DeleteIvr(t Ivr) error {
	os.Remove(filepath.Join(db.filePath, "Ivr", "Ivr_"+t.IvrID+".kcfg"))
	return db.mem_engine.DeleteIvr(t)
}

func (db *kcfgStore) CreateTeleGramUser(TeleGramUser) error {
	return nil
}

func (db *kcfgStore) StoreTeleGramUser(t TeleGramUser) error {
	return db.mem_engine.StoreTeleGramUser(t)
}

func (db *kcfgStore) QueryTeleGramUser(t TeleGramUser) (TeleGramUser, error) {
	return db.mem_engine.QueryTeleGramUser(t)
}

func (db *kcfgStore) CountTeleGramUser() int64 {
	return db.mem_engine.CountTeleGramUser()
}

func (db *kcfgStore) QueryTeleGramUserOfPage(offset int64, size int64) ([]TeleGramUser, error) {
	return db.mem_engine.QueryTeleGramUserOfPage(offset, size)
}

func (db *kcfgStore) DeleteTeleGramUser(t TeleGramUser) error {
	return db.mem_engine.DeleteTeleGramUser(t)
}

func (db *kcfgStore) Unload() error {
	return db.mem_engine.Unload()
}

func init() {
	engines["file://"] = &kcfgStore{}
}
