package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitee.com/kolonse_zhjsh/gpbx/kcfg"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const ThisModuleGorm = "GormStore"

type gormStore struct {
	logger   log.Logger
	gormDB   *gorm.DB
	filePath string
}

func (db *gormStore) SetLogger(logger log.Logger) {
	db.logger = logger
}

func (db *gormStore) Load(path string) error {
	// 解析路径: sqlite://path/to/gpbx.db
	dbPath := path[9:] // 去掉 "sqlite://"
	if dbPath == "" {
		dbPath = "gpbx.db"
	}

	// 确保数据库文件所在目录存在
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			db.logger.Error(ThisModuleGorm, "Create DB Dir Failed, err:%v", err)
			return err
		}
	}

	db.filePath = dbPath

	var err error
	db.gormDB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		db.logger.Error(ThisModuleGorm, "Failed to open database: %v", err)
		return err
	}

	// 配置 SQLite 连接池与性能参数
	sqlDB, err := db.gormDB.DB()
	if err != nil {
		db.logger.Error(ThisModuleGorm, "Failed to get sql.DB: %v", err)
		return err
	}
	sqlDB.SetMaxOpenConns(1) // SQLite 单写者模式
	sqlDB.SetMaxIdleConns(1)

	// 启用 WAL 模式，提高并发读性能
	db.gormDB.Exec("PRAGMA journal_mode=WAL")
	db.gormDB.Exec("PRAGMA busy_timeout=5000")
	db.gormDB.Exec("PRAGMA synchronous=NORMAL")

	// 自动迁移所有表
	err = db.gormDB.AutoMigrate(&User{}, &Tenant{}, &TenantNumber{}, &Robot{}, &Extension{}, &Agent{}, &Ivr{}, &TeleGramUser{})
	if err != nil {
		db.logger.Error(ThisModuleGorm, "Failed to auto migrate: %v", err)
		return err
	}

	// 手动创建唯一索引（SQLite AutoMigrate 对 uniqueIndex 支持有限）
	db.gormDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_extension_id ON extensions(extension_id)")
	db.gormDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_id ON agents(agent_id)")

	db.logger.Info(ThisModuleGorm, "Database opened: %s", dbPath)

	// 自动检测并迁移 kcfg 数据
	dbDir := filepath.Dir(dbPath)
	if dbDir != "." && dbDir != "" {
		db.migrateFromKcfg(dbDir)
	}

	return nil
}

func (db *gormStore) Unload() error {
	if db.gormDB != nil {
		sqlDB, err := db.gormDB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// ==================== kcfg 数据迁移 ====================

// hasKcfgData 检查目录中是否存在 kcfg 数据子目录
func (db *gormStore) hasKcfgData(dir string) bool {
	subDirs := []string{"User", "Tenant", "TenantNumber", "Robot", "Extension", "Agent", "Ivr"}
	for _, sub := range subDirs {
		info, err := os.Stat(filepath.Join(dir, sub))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// isDatabaseEmpty 检查数据库是否为空（无任何业务数据）
func (db *gormStore) isDatabaseEmpty() bool {
	var count int64
	db.gormDB.Model(&User{}).Count(&count)
	if count > 0 {
		return false
	}
	db.gormDB.Model(&Tenant{}).Count(&count)
	if count > 0 {
		return false
	}
	db.gormDB.Model(&Robot{}).Count(&count)
	if count > 0 {
		return false
	}
	db.gormDB.Model(&Ivr{}).Count(&count)
	return count == 0
}

// migrateFromKcfg 从 kcfg 文件目录迁移数据到 SQLite
// 仅在数据库为空且 kcfg 目录存在时执行，迁移后不删除原始文件
func (db *gormStore) migrateFromKcfg(kcfgDir string) {
	// 1. 检查是否存在 kcfg 数据目录
	if !db.hasKcfgData(kcfgDir) {
		return
	}

	// 2. 检查数据库是否为空
	if !db.isDatabaseEmpty() {
		db.logger.Info(ThisModuleGorm, "Database is not empty, skip kcfg migration")
		return
	}

	db.logger.Info(ThisModuleGorm, "Detected kcfg data in %s, start migrating...", kcfgDir)

	// 3. 遍历目录，解析并迁移每个 kcfg 文件
	filepath.WalkDir(kcfgDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".kcfg" {
			return nil
		}
		filename := filepath.Base(path)
		switch {
		case strings.HasPrefix(filename, "User_"):
			db.migrateUserKcfg(path)
		case strings.HasPrefix(filename, "Tenant_"):
			db.migrateTenantKcfg(path)
		case strings.HasPrefix(filename, "TenantNumber_"):
			db.migrateTenantNumberKcfg(path)
		case strings.HasPrefix(filename, "Robot_"):
			db.migrateRobotKcfg(path)
		case strings.HasPrefix(filename, "Extension_"):
			db.migrateExtensionKcfg(path)
		case strings.HasPrefix(filename, "Agent_"):
			db.migrateAgentKcfg(path)
		case strings.HasPrefix(filename, "Ivr_"):
			db.migrateIvrKcfg(path)
		}
		return nil
	})

	db.logger.Info(ThisModuleGorm, "kcfg migration completed")
}

func (db *gormStore) migrateUserKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate User kcfg %s failed: %v", file, r)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	user := User{
		ID:        uint(cfg.Child("user.ID").GetInt()),
		Username:  cfg.Child("user.Username").GetString(),
		Password:  cfg.Child("user.Password").GetString(), // kcfg 中已是 bcrypt 哈希，不重新加密
		Name:      cfg.Child("user.Name").GetString(),
		Roles:     cfg.Child("user.Roles").GetString(),
		CreatedAt: cfg.Child("user.CreateTime").GetInt(),
		UpdatedAt: cfg.Child("user.UpdateTime").GetInt(),
	}
	// 直接插入数据库，跳过 StoreUser 的密码加密逻辑（密码已是哈希值）
	if err := db.gormDB.Create(&user).Error; err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate User %s failed: %v", user.Username, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated User: %s", user.Username)
	}
}

func (db *gormStore) migrateTenantKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate Tenant kcfg %s failed: %v", file, r)
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
	if err := db.StoreTenant(tenant); err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate Tenant %s failed: %v", tenant.TenantId, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated Tenant: %s", tenant.TenantId)
	}
}

func (db *gormStore) migrateTenantNumberKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate TenantNumber kcfg %s failed: %v", file, r)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	tn := TenantNumber{
		Number:     cfg.Child("tenant_number.Number").GetString(),
		TenantId:   cfg.Child("tenant_number.TenantId").GetString(),
		CreateTime: cfg.Child("tenant_number.CreateTime").GetInt(),
		ExpireTime: cfg.Child("tenant_number.ExpireTime").GetInt(),
		Action:     cfg.Child("tenant_number.Action").GetInt(),
		WayType:    cfg.Child("tenant_number.WayType").GetString(),
		Way:        cfg.Child("tenant_number.Way").GetString(),
		RobotID:    cfg.Child("tenant_number.RobotID").GetString(),
	}
	if err := db.StoreTenantNumber(tn); err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate TenantNumber %s failed: %v", tn.Number, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated TenantNumber: %s", tn.Number)
	}
}

func (db *gormStore) migrateRobotKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate Robot kcfg %s failed: %v", file, r)
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
	// Arg 字段是 JSON 文本，需反序列化
	argStr := cfg.Child("robot.Arg").GetString()
	if argStr != "" {
		if err := json.Unmarshal([]byte(argStr), &robot.Arg); err != nil {
			db.logger.Warn(ThisModuleGorm, "Migrate Robot %s Arg parse failed: %v", robot.RobotID, err)
			robot.Arg = make(map[string]any)
		}
	}
	if err := db.StoreRobot(robot); err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate Robot %s failed: %v", robot.RobotID, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated Robot: %s", robot.RobotID)
	}
}

func (db *gormStore) migrateExtensionKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate Extension kcfg %s failed: %v", file, r)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	ext := Extension{
		TenantId:    cfg.Child("extension.TenantId").GetString(),
		ExtensionId: cfg.Child("extension.ExtensionId").GetString(),
		CreateTime:  cfg.Child("extension.CreateTime").GetInt(),
		Status:      cfg.Child("extension.status").GetString(),
		NetworkIP:   cfg.Child("extension.networkIp").GetString(),
		NetworkPort: cfg.Child("extension.networkPort").GetString(),
	}
	if err := db.StoreExtension(ext); err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate Extension %s failed: %v", ext.ExtensionId, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated Extension: %s", ext.ExtensionId)
	}
}

func (db *gormStore) migrateAgentKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate Agent kcfg %s failed: %v", file, r)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(file)

	agent := Agent{
		TenantId:    cfg.Child("agent.TenantId").GetString(),
		AgentId:     cfg.Child("agent.AgentId").GetString(),
		AgentName:   cfg.Child("agent.AgentName").GetString(),
		ExtensionId: cfg.Child("agent.ExtensionId").GetString(),
		CreateTime:  cfg.Child("agent.CreateTime").GetInt(),
	}
	if err := db.StoreAgent(agent); err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate Agent %s failed: %v", agent.AgentId, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated Agent: %s", agent.AgentId)
	}
}

func (db *gormStore) migrateIvrKcfg(file string) {
	defer func() {
		if r := recover(); r != nil {
			db.logger.Error(ThisModuleGorm, "Migrate Ivr kcfg %s failed: %v", file, r)
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
	if err := db.StoreIvr(ivr); err != nil {
		db.logger.Error(ThisModuleGorm, "Migrate Ivr %s failed: %v", ivr.IvrID, err)
	} else {
		db.logger.Info(ThisModuleGorm, "Migrated Ivr: %s", ivr.IvrID)
	}
}

// ==================== User ====================

func (db *gormStore) CreateUser(User) error {
	return nil // 表已通过 AutoMigrate 自动创建
}

func (db *gormStore) StoreUser(u User) error {
	var existing User
	result := db.gormDB.Where("username = ?", u.Username).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		// 新建用户，手动加密密码（GORM hook 签名不兼容，不自动触发）
		if u.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			u.Password = string(hashedPassword)
		}
		return db.gormDB.Create(&u).Error
	}
	if result.Error != nil {
		return result.Error
	}

	// 仅更新非空字段（使用 map 方式不触发方法，需手动加密密码）
	updates := map[string]any{}
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		updates["password"] = string(hashedPassword)
	}
	if u.Name != "" {
		updates["name"] = u.Name
	}
	if u.Roles != "" {
		updates["roles"] = u.Roles
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryUser(u User) (User, error) {
	var result User
	err := db.gormDB.Where(&u).First(&result).Error
	if err != nil {
		return User{}, err
	}
	return result, nil
}

func (db *gormStore) CountUser() int64 {
	var count int64
	db.gormDB.Model(&User{}).Count(&count)
	return count
}

// ==================== Tenant ====================

func (db *gormStore) CreateTenant(Tenant) error {
	return nil
}

func (db *gormStore) StoreTenant(t Tenant) error {
	var existing Tenant
	result := db.gormDB.Where("tenant_id = ?", t.TenantId).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return db.gormDB.Create(&t).Error
	}
	if result.Error != nil {
		return result.Error
	}

	updates := map[string]any{}
	if t.TenantName != "" {
		updates["tenant_name"] = t.TenantName
	}
	if t.CreateTime > 0 {
		updates["create_time"] = t.CreateTime
	}
	if t.ExpireTime > 0 {
		updates["expire_time"] = t.ExpireTime
	}
	if t.DefaultNumber != "" {
		updates["default_number"] = t.DefaultNumber
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryTenant(t Tenant) (Tenant, error) {
	var result Tenant
	err := db.gormDB.Where(&t).First(&result).Error
	if err != nil {
		return Tenant{}, err
	}
	return result, nil
}

func (db *gormStore) CountTenant() int64 {
	var count int64
	db.gormDB.Model(&Tenant{}).Count(&count)
	return count
}

func (db *gormStore) QueryTenantOfPage(offset int64, size int64) ([]Tenant, error) {
	var results []Tenant
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteTenant(t Tenant) error {
	return db.gormDB.Where("tenant_id = ?", t.TenantId).Delete(&Tenant{}).Error
}

// ==================== TenantNumber ====================

func (db *gormStore) CreateTenantNumber(TenantNumber) error {
	return nil
}

func (db *gormStore) StoreTenantNumber(t TenantNumber) error {
	var existing TenantNumber
	result := db.gormDB.Where("number = ?", t.Number).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return db.gormDB.Create(&t).Error
	}
	if result.Error != nil {
		return result.Error
	}

	updates := map[string]any{}
	if t.TenantId != "" {
		updates["tenant_id"] = t.TenantId
	}
	if t.CreateTime > 0 {
		updates["create_time"] = t.CreateTime
	}
	if t.ExpireTime > 0 {
		updates["expire_time"] = t.ExpireTime
	}
	if t.Action != 0 {
		updates["action"] = t.Action
	}
	if t.WayType != "" {
		updates["way_type"] = t.WayType
	}
	if t.Way != "" {
		updates["way"] = t.Way
	}
	if t.RobotID != "" {
		updates["robot_id"] = t.RobotID
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryTenantNumber(t TenantNumber) (TenantNumber, error) {
	var result TenantNumber
	err := db.gormDB.Where(&t).First(&result).Error
	if err != nil {
		return TenantNumber{}, err
	}
	return result, nil
}

// QueryTenantNumbers 查询租户号码列表，支持多条件组合查询（AND关系）
func (db *gormStore) QueryTenantNumbers(t TenantNumber) ([]TenantNumber, error) {
	var results []TenantNumber
	err := db.gormDB.Where(&t).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) CountTenantNumber() int64 {
	var count int64
	db.gormDB.Model(&TenantNumber{}).Count(&count)
	return count
}

func (db *gormStore) QueryTenantNumberOfPage(offset int64, size int64) ([]TenantNumber, error) {
	var results []TenantNumber
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteTenantNumber(t TenantNumber) error {
	return db.gormDB.Where("number = ?", t.Number).Delete(&TenantNumber{}).Error
}

// ==================== Robot ====================

func (db *gormStore) CreateRobot(Robot) error {
	return nil
}

func (db *gormStore) StoreRobot(t Robot) error {
	var existing Robot
	result := db.gormDB.Where("robot_id = ?", t.RobotID).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return db.gormDB.Create(&t).Error
	}
	if result.Error != nil {
		return result.Error
	}

	updates := map[string]any{}
	if t.Target != "" {
		updates["target"] = t.Target
	}
	if t.Arg != nil {
		// Updates 使用 map 时 GORM serializer 不生效，需手动 JSON 序列化
		argBytes, err := json.Marshal(t.Arg)
		if err != nil {
			return err
		}
		updates["arg"] = string(argBytes)
	}
	if t.Welcome != "" {
		updates["welcome"] = t.Welcome
	}
	if t.Prompt != "" {
		updates["prompt"] = t.Prompt
	}
	if t.CreateTime > 0 {
		updates["create_time"] = t.CreateTime
	}
	// ToVendor 是 bool 类型，需要特殊处理：即使为 false 也应能更新
	updates["to_vendor"] = t.ToVendor
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryRobot(t Robot) (Robot, error) {
	var result Robot
	err := db.gormDB.Where(&t).First(&result).Error
	if err != nil {
		return Robot{}, err
	}
	return result, nil
}

func (db *gormStore) CountRobot() int64 {
	var count int64
	db.gormDB.Model(&Robot{}).Count(&count)
	return count
}

func (db *gormStore) QueryRobotOfPage(offset int64, size int64) ([]Robot, error) {
	var results []Robot
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteRobot(t Robot) error {
	return db.gormDB.Where("robot_id = ?", t.RobotID).Delete(&Robot{}).Error
}

// ==================== Extension ====================

func (db *gormStore) CreateExtension(Extension) error {
	return nil
}

func (db *gormStore) StoreExtension(e Extension) error {
	var existing Extension
	result := db.gormDB.Where("extension_id = ? and tenant_id = ?", e.ExtensionId, e.TenantId).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		// 插入时确保 Status 默认为 offline
		if e.Status == "" {
			e.Status = "offline"
		}
		return db.gormDB.Create(&e).Error
	}
	if result.Error != nil {
		return result.Error
	}

	// 更新时必须确保主键有值
	if e.TenantId == "" || e.ExtensionId == "" {
		return fmt.Errorf("StoreExtension: tenant_id and extension_id are required for update")
	}

	updates := map[string]any{}
	if e.CreateTime > 0 {
		updates["create_time"] = e.CreateTime
	}
	e.HashPassword()
	if e.Password != "" {
		updates["password"] = e.Password
	}
	if e.Status != "" {
		updates["status"] = e.Status
	}
	if e.NetworkIP != "" {
		updates["network_ip"] = e.NetworkIP
	}
	if e.NetworkPort != "" {
		updates["network_port"] = e.NetworkPort
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryExtension(e Extension) (Extension, error) {
	var result Extension
	err := db.gormDB.Where(&e).First(&result).Error
	if err != nil {
		return Extension{}, err
	}
	return result, nil
}

func (db *gormStore) CountExtension() int64 {
	var count int64
	db.gormDB.Model(&Extension{}).Count(&count)
	return count
}

func (db *gormStore) QueryExtensionOfPage(offset int64, size int64) ([]Extension, error) {
	var results []Extension
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteExtension(e Extension) error {
	query := db.gormDB.Where("extension_id = ?", e.ExtensionId)
	if e.TenantId != "" {
		query = query.Where("tenant_id = ?", e.TenantId)
	}
	return query.Delete(&Extension{}).Error
}

// ==================== Agent ====================

func (db *gormStore) CreateAgent(Agent) error {
	return nil
}

func (db *gormStore) StoreAgent(a Agent) error {
	var existing Agent
	result := db.gormDB.Where("agent_id = ? and tenant_id = ?", a.AgentId, a.TenantId).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return db.gormDB.Create(&a).Error
	}
	if result.Error != nil {
		return result.Error
	}

	updates := map[string]any{}
	if a.TenantId != "" {
		updates["tenant_id"] = a.TenantId
	}
	if a.AgentName != "" {
		updates["agent_name"] = a.AgentName
	}
	if a.ExtensionId != "" {
		updates["extension_id"] = a.ExtensionId
	}
	if a.CreateTime > 0 {
		updates["create_time"] = a.CreateTime
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryAgent(a Agent) (Agent, error) {
	var result Agent
	err := db.gormDB.Where(&a).First(&result).Error
	if err != nil {
		return Agent{}, err
	}
	return result, nil
}

func (db *gormStore) CountAgent() int64 {
	var count int64
	db.gormDB.Model(&Agent{}).Count(&count)
	return count
}

func (db *gormStore) QueryAgentOfPage(offset int64, size int64) ([]Agent, error) {
	var results []Agent
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteAgent(a Agent) error {
	query := db.gormDB.Where("agent_id = ?", a.AgentId)
	if a.TenantId != "" {
		query = query.Where("tenant_id = ?", a.TenantId)
	}
	return query.Delete(&Agent{}).Error
}

// ==================== Ivr ====================

func (db *gormStore) CreateIvr(Ivr) error {
	return nil
}

func (db *gormStore) StoreIvr(t Ivr) error {
	var existing Ivr
	result := db.gormDB.Where("ivr_id = ?", t.IvrID).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return db.gormDB.Create(&t).Error
	}
	if result.Error != nil {
		return result.Error
	}

	updates := map[string]any{}
	if t.Type != "" {
		updates["type"] = t.Type
	}
	if t.Path != "" {
		updates["path"] = t.Path
	}
	if t.Args != "" {
		updates["args"] = t.Args
	}
	if t.CreateTime > 0 {
		updates["create_time"] = t.CreateTime
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryIvr(t Ivr) (Ivr, error) {
	var result Ivr
	err := db.gormDB.Where(&t).First(&result).Error
	if err != nil {
		return Ivr{}, err
	}
	return result, nil
}

func (db *gormStore) CountIvr() int64 {
	var count int64
	db.gormDB.Model(&Ivr{}).Count(&count)
	return count
}

func (db *gormStore) QueryIvrOfPage(offset int64, size int64) ([]Ivr, error) {
	var results []Ivr
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteIvr(t Ivr) error {
	return db.gormDB.Where("ivr_id = ?", t.IvrID).Delete(&Ivr{}).Error
}

// ==================== TeleGramUser ====================

func (db *gormStore) CreateTeleGramUser(TeleGramUser) error {
	return nil
}

func (db *gormStore) StoreTeleGramUser(t TeleGramUser) error {
	var existing TeleGramUser
	result := db.gormDB.Where("username = ?", t.Username).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return db.gormDB.Create(&t).Error
	}
	if result.Error != nil {
		return result.Error
	}

	updates := map[string]any{}
	if t.TenantId != "" {
		updates["tenant_id"] = t.TenantId
	}
	if t.AuthTime > 0 {
		updates["auth_time"] = t.AuthTime
	}
	if t.ExpireTime > 0 {
		updates["expire_time"] = t.ExpireTime
	}
	if t.BindScript != "" {
		updates["bind_script"] = t.BindScript
	}
	if t.BindNumbers != "" {
		updates["bind_numbers"] = t.BindNumbers
	}
	if t.IvrId != "" {
		updates["ivr_id"] = t.IvrId
	}
	if len(updates) > 0 {
		return db.gormDB.Model(&existing).Updates(updates).Error
	}
	return nil
}

func (db *gormStore) QueryTeleGramUser(t TeleGramUser) (TeleGramUser, error) {
	var result TeleGramUser
	q := db.gormDB
	if t.Username != "" {
		q = q.Where("username = ?", t.Username)
	}
	if t.TenantId != "" {
		q = q.Where("tenant_id = ?", t.TenantId)
	}
	err := q.First(&result).Error
	if err != nil {
		return TeleGramUser{}, err
	}
	return result, nil
}

func (db *gormStore) CountTeleGramUser() int64 {
	var count int64
	db.gormDB.Model(&TeleGramUser{}).Count(&count)
	return count
}

func (db *gormStore) QueryTeleGramUserOfPage(offset int64, size int64) ([]TeleGramUser, error) {
	var results []TeleGramUser
	err := db.gormDB.Offset(int(offset)).Limit(int(size)).Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (db *gormStore) DeleteTeleGramUser(t TeleGramUser) error {
	return db.gormDB.Where("username = ?", t.Username).Delete(&TeleGramUser{}).Error
}

// ==================== init ====================

func init() {
	engines["sqlite://"] = &gormStore{}
}
