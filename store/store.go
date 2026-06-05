package store

import (
	"strings"

	"gitee.com/kolonse_zhjsh/gpbx/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	NUMBER_ACTION_to_robot  = 1
	NUMBER_ACTION_to_vendor = 2
)

const (
	WAY_TYPE_gateway = "gateway" // 网关类型
	WAY_TYPE_ims     = "ims"     // ims 类型，需要路由网关地址
	WAY_TYPE_local   = "local"   // 本地用户
)

// IVR 类型枚举
const (
	IVR_TYPE_Lua  = "lua"  // Lua 脚本类型
	IVR_TYPE_Dify = "dify" // Dify 工作流类型
)

// User 用户
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey;comment:用户ID"`
	Username  string         `json:"username" gorm:"size:50;uniqueIndex;not null;comment:用户名"`
	Password  string         `json:"-" gorm:"size:100;not null;comment:密码"`
	Name      string         `json:"name" gorm:"size:50;not null;comment:姓名"`
	Roles     string         `json:"roles" gorm:"size:100;default:'user';comment:角色"`
	CreatedAt int64          `json:"created_at" gorm:"autoCreateTime:milli;comment:创建时间"`
	UpdatedAt int64          `json:"updated_at" gorm:"autoUpdateTime:milli;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

func (u *User) HashPassword() error {
	// 如果密码非空，则加密
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// Tenant 租户
type Tenant struct {
	TenantId      string `json:"tenantId" gorm:"primaryKey;size:50;comment:租户ID"`
	TenantName    string `json:"tenantName" gorm:"size:50;not null;comment:租户名称"`
	CreateTime    int64  `json:"createTime" gorm:"comment:创建时间"`
	ExpireTime    int64  `json:"expireTime" gorm:"comment:过期时间"`
	DefaultNumber string `json:"default_number" gorm:"size:50;default:'';comment:默认号码"`
}

// TenantNumber 租户号码
type TenantNumber struct {
	Number     string `json:"number" gorm:"primaryKey;size:50;comment:号码"`
	TenantId   string `json:"tenantId" gorm:"size:50;not null;index;comment:租户ID"`
	CreateTime int64  `json:"createTime" gorm:"comment:创建时间"`
	ExpireTime int64  `json:"expireTime" gorm:"comment:过期时间"`
	Action     int64  `json:"action" gorm:"default:0;comment:动作类型(1:转机器人 2:转三方)"`
	WayType    string `json:"wayType" gorm:"size:20;comment:路由类型(gateway/ims/local)"`
	Way        string `json:"way" gorm:"size:100;comment:路由地址"`
	RobotID    string `json:"robotId" gorm:"size:50;comment:机器人ID"`
}

// Robot 机器人
type Robot struct {
	RobotID    string         `json:"robotId" gorm:"primaryKey;size:50;comment:机器人ID"`
	Target     string         `json:"target" gorm:"size:200;comment:目标地址"`
	Arg        map[string]any `json:"arg" gorm:"type:text;serializer:json;comment:参数"`
	Welcome    string         `json:"welcome" gorm:"type:text;comment:欢迎语"`
	Prompt     string         `json:"prompt" gorm:"type:text;comment:提示词"`
	CreateTime int64          `json:"createTime" gorm:"comment:创建时间"`
	ToVendor   bool           `json:"toVendor" gorm:"default:false;comment:是否三方供应商"`
}

const (
	EXTENSION_STATUS_ONLINE  = "online"
	EXTENSION_STATUS_OFFLINE = "offline"
)

// Extension 分机
type Extension struct {
	TenantId    string `json:"tenantId" gorm:"primaryKey;size:50;not null;index;comment:租户ID"`
	ExtensionId string `json:"extensionId" gorm:"primaryKey;size:50;uniqueIndex;comment:分机ID"`
	Password    string `json:"password" gorm:"size:50;comment:密码"`
	CreateTime  int64  `json:"createTime" gorm:"comment:创建时间"`
	Status      string `json:"status" gorm:"size:10;default:'offline';comment:在线状态(online/offline)"`
	NetworkIP   string `json:"networkIp" gorm:"size:50;default:'';comment:分机IP地址"`
	NetworkPort string `json:"networkPort" gorm:"size:20;default:'';comment:分机端口"`
}

func (u *Extension) HashPassword() error {
	// 如果密码非空，则加密
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

func (u *Extension) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// Agent 坐席
type Agent struct {
	TenantId      string `json:"tenantId" gorm:"primaryKey;size:50;not null;index;comment:租户ID"`
	AgentId       string `json:"agentId" gorm:"primaryKey;size:50;uniqueIndex;comment:坐席ID"`
	AgentName     string `json:"agentName" gorm:"size:50;not null;comment:坐席名称"`
	ExtensionId   string `json:"extensionId" gorm:"size:50;not null;index;comment:分机ID"`
	DisplayNumber string `json:"displayNumber" gorm:"size:50;comment:外显号码"`
	CreateTime    int64  `json:"createTime" gorm:"comment:创建时间"`
}

// Ivr IVR 配置
type Ivr struct {
	IvrID      string `json:"ivrId" gorm:"primaryKey;size:50;comment:IVR ID"`
	Type       string `json:"type" gorm:"size:20;comment:类型(Lua/Dify)"`
	Path       string `json:"path" gorm:"size:500;comment:路径"`
	Args       string `json:"args" gorm:"type:text;comment:参数(JSON格式，Type为Dify时有效)"`
	CreateTime int64  `json:"createTime" gorm:"comment:创建时间"`
}

// TeleGramUser Telegram 用户
type TeleGramUser struct {
	Username    string `json:"username" gorm:"primaryKey;size:100;comment:用户名"`
	TenantId    string `json:"tenantId" gorm:"size:50;not null;index;comment:租户ID"`
	AuthTime    int64  `json:"authTime" gorm:"comment:授权时间"`
	ExpireTime  int64  `json:"expireTime" gorm:"comment:过期时间"`
	BindScript  string `json:"bindScript" gorm:"size:500;not null;comment:绑定脚本"`
	BindNumbers string `json:"bindNumbers" gorm:"size:500;default:'*';comment:绑定号码，*表示绑定所有，格式 号码1;号码2;号码3"`
	IvrId       string `json:"ivrId" gorm:"size:50;default:'';comment:IVR ID"`
}

// IsExpired 判断是否过期：过期时间 < 授权时间，则返回过期
func (u *TeleGramUser) IsExpired() bool {
	return u.ExpireTime < u.AuthTime
}

type Store interface {
	Load(string) error
	SetLogger(log.Logger)

	CreateUser(User) error
	StoreUser(User) error
	QueryUser(User) (User, error)
	CountUser() int64

	CreateTenant(Tenant) error
	StoreTenant(Tenant) error
	QueryTenant(Tenant) (Tenant, error)
	CountTenant() int64
	QueryTenantOfPage(offset int64, size int64) ([]Tenant, error)
	DeleteTenant(Tenant) error

	CreateTenantNumber(TenantNumber) error
	StoreTenantNumber(TenantNumber) error
	QueryTenantNumber(TenantNumber) (TenantNumber, error)
	QueryTenantNumbers(TenantNumber) ([]TenantNumber, error)
	CountTenantNumber() int64
	QueryTenantNumberOfPage(offset int64, size int64) ([]TenantNumber, error)
	DeleteTenantNumber(TenantNumber) error

	CreateRobot(Robot) error
	StoreRobot(Robot) error
	QueryRobot(Robot) (Robot, error)
	CountRobot() int64
	QueryRobotOfPage(offset int64, size int64) ([]Robot, error)
	DeleteRobot(Robot) error

	CreateExtension(Extension) error
	StoreExtension(Extension) error
	QueryExtension(Extension) (Extension, error)
	CountExtension() int64
	QueryExtensionOfPage(offset int64, size int64) ([]Extension, error)
	DeleteExtension(Extension) error

	CreateAgent(Agent) error
	StoreAgent(Agent) error
	QueryAgent(Agent) (Agent, error)
	CountAgent() int64
	QueryAgentOfPage(offset int64, size int64) ([]Agent, error)
	DeleteAgent(Agent) error

	CreateIvr(Ivr) error
	StoreIvr(Ivr) error
	QueryIvr(Ivr) (Ivr, error)
	CountIvr() int64
	QueryIvrOfPage(offset int64, size int64) ([]Ivr, error)
	DeleteIvr(Ivr) error

	CreateTeleGramUser(TeleGramUser) error
	StoreTeleGramUser(TeleGramUser) error
	QueryTeleGramUser(TeleGramUser) (TeleGramUser, error)
	CountTeleGramUser() int64
	QueryTeleGramUserOfPage(offset int64, size int64) ([]TeleGramUser, error)
	DeleteTeleGramUser(TeleGramUser) error

	Unload() error
}

func Get(path string) Store {
	for k, engine := range engines {
		if strings.Index(path, k) == 0 {
			return engine
		}
	}
	return nil
}

var engines map[string]Store = make(map[string]Store)

// GetEngines 返回所有已注册的 Store 引擎（用于扩展 Store 接口）
func GetEngines() map[string]Store {
	return engines
}

// SetEngine 替换指定路径前缀的 Store 引擎
func SetEngine(path string, engine Store) {
	engines[path] = engine
}
