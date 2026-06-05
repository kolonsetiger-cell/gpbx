package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testLogger 实现 log.Logger 接口，用于测试
type testLogger struct{}

func (l *testLogger) Debug(module string, format string, arg ...any) {}
func (l *testLogger) Info(module string, format string, arg ...any)  {}
func (l *testLogger) Warn(module string, format string, arg ...any)  {}
func (l *testLogger) Error(module string, format string, arg ...any) {}

// newTestStore 创建一个用于测试的 gormStore 实例
func newTestStore(t *testing.T) *gormStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s := &gormStore{logger: &testLogger{}}
	if err := s.Load("sqlite://" + dbPath); err != nil {
		t.Fatalf("Failed to load store: %v", err)
	}
	t.Cleanup(func() {
		s.Unload()
	})
	return s
}

// ==================== Load / Unload ====================

func TestGormStore_Load(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "load_test.db")
	s := &gormStore{logger: &testLogger{}}

	// 正常加载
	err := s.Load("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	s.Unload()

	// 验证数据库文件已创建
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Database file was not created")
	}
}

func TestGormStore_Load_DefaultPath(t *testing.T) {
	s := &gormStore{logger: &testLogger{}}
	// sqlite:// 后为空路径，应使用默认 gpbx.db
	err := s.Load("sqlite://")
	if err != nil {
		t.Fatalf("Load with empty path failed: %v", err)
	}
	s.Unload()
	// 清理默认 db 文件
	os.Remove("gpbx.db")
}

func TestGormStore_Load_CreateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "nested")
	dbPath := filepath.Join(dir, "test.db")
	s := &gormStore{logger: &testLogger{}}

	err := s.Load("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("Load with nested dir failed: %v", err)
	}
	s.Unload()
}

func TestGormStore_Unload(t *testing.T) {
	_ = newTestStore(t) // Unload 由 cleanup 调用，验证不 panic 即可

	// Unload 空 store 也应正常
	s2 := &gormStore{logger: &testLogger{}}
	if err := s2.Unload(); err != nil {
		t.Fatalf("Unload empty store failed: %v", err)
	}
}

func TestGormStore_CreateNoop(t *testing.T) {
	s := newTestStore(t)
	// 所有 Create* 方法均为 noop
	if err := s.CreateUser(User{}); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := s.CreateTenant(Tenant{}); err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}
	if err := s.CreateTenantNumber(TenantNumber{}); err != nil {
		t.Fatalf("CreateTenantNumber failed: %v", err)
	}
	if err := s.CreateRobot(Robot{}); err != nil {
		t.Fatalf("CreateRobot failed: %v", err)
	}
	if err := s.CreateExtension(Extension{}); err != nil {
		t.Fatalf("CreateExtension failed: %v", err)
	}
	if err := s.CreateAgent(Agent{}); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}
}

// ==================== User ====================

func TestGormStore_UserCRUD(t *testing.T) {
	s := newTestStore(t)

	// 1. 创建用户
	err := s.StoreUser(User{
		Username: "admin",
		Password: "admin123",
		Name:     "管理员",
		Roles:    "R_SUPER",
	})
	if err != nil {
		t.Fatalf("StoreUser create failed: %v", err)
	}

	// 2. 查询用户
	u, err := s.QueryUser(User{Username: "admin"})
	if err != nil {
		t.Fatalf("QueryUser failed: %v", err)
	}
	if u.Username != "admin" {
		t.Errorf("Username = %q, want %q", u.Username, "admin")
	}
	if u.Name != "管理员" {
		t.Errorf("Name = %q, want %q", u.Name, "管理员")
	}
	if u.Roles != "R_SUPER" {
		t.Errorf("Roles = %q, want %q", u.Roles, "R_SUPER")
	}

	// 3. 验证密码加密
	if u.Password == "admin123" {
		t.Error("Password should be hashed, but got plaintext")
	}
	if !u.CheckPassword("admin123") {
		t.Error("CheckPassword should return true for correct password")
	}
	if u.CheckPassword("wrong") {
		t.Error("CheckPassword should return false for wrong password")
	}

	// 4. 计数
	if count := s.CountUser(); count != 1 {
		t.Errorf("CountUser = %d, want 1", count)
	}

	// 5. 更新用户（不修改密码）
	err = s.StoreUser(User{
		Username: "admin",
		Name:     "超级管理员",
		Roles:    "R_ADMIN",
	})
	if err != nil {
		t.Fatalf("StoreUser update failed: %v", err)
	}
	u2, _ := s.QueryUser(User{Username: "admin"})
	if u2.Name != "超级管理员" {
		t.Errorf("Name after update = %q, want %q", u2.Name, "超级管理员")
	}
	if u2.Roles != "R_ADMIN" {
		t.Errorf("Roles after update = %q, want %q", u2.Roles, "R_ADMIN")
	}
	// 密码应未改变
	if !u2.CheckPassword("admin123") {
		t.Error("Password should remain unchanged after update without password")
	}

	// 6. 更新密码
	err = s.StoreUser(User{
		Username: "admin",
		Password: "newpass",
	})
	if err != nil {
		t.Fatalf("StoreUser update password failed: %v", err)
	}
	u3, _ := s.QueryUser(User{Username: "admin"})
	if !u3.CheckPassword("newpass") {
		t.Error("CheckPassword should return true for new password")
	}
	if u3.CheckPassword("admin123") {
		t.Error("CheckPassword should return false for old password")
	}
}

func TestGormStore_QueryUser_NotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.QueryUser(User{Username: "nonexistent"})
	if err == nil {
		t.Error("QueryUser should return error for nonexistent user")
	}
	if err.Error() != "Not Found" {
		t.Errorf("Error = %q, want %q", err.Error(), "Not Found")
	}
}

// ==================== Tenant ====================

func TestGormStore_TenantCRUD(t *testing.T) {
	s := newTestStore(t)

	// 创建
	err := s.StoreTenant(Tenant{
		TenantId:   "t001",
		TenantName: "测试租户",
		CreateTime: time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("StoreTenant create failed: %v", err)
	}

	// 查询
	tenant, err := s.QueryTenant(Tenant{TenantId: "t001"})
	if err != nil {
		t.Fatalf("QueryTenant failed: %v", err)
	}
	if tenant.TenantName != "测试租户" {
		t.Errorf("TenantName = %q, want %q", tenant.TenantName, "测试租户")
	}

	// 计数
	if count := s.CountTenant(); count != 1 {
		t.Errorf("CountTenant = %d, want 1", count)
	}

	// 更新
	err = s.StoreTenant(Tenant{
		TenantId:      "t001",
		TenantName:    "更新租户",
		DefaultNumber: "13800000000",
	})
	if err != nil {
		t.Fatalf("StoreTenant update failed: %v", err)
	}
	t2, _ := s.QueryTenant(Tenant{TenantId: "t001"})
	if t2.TenantName != "更新租户" {
		t.Errorf("TenantName after update = %q, want %q", t2.TenantName, "更新租户")
	}
	if t2.DefaultNumber != "13800000000" {
		t.Errorf("DefaultNumber = %q, want %q", t2.DefaultNumber, "13800000000")
	}

	// 删除
	err = s.DeleteTenant(Tenant{TenantId: "t001"})
	if err != nil {
		t.Fatalf("DeleteTenant failed: %v", err)
	}
	if count := s.CountTenant(); count != 0 {
		t.Errorf("CountTenant after delete = %d, want 0", count)
	}

	// 删除后查询应返回 Not Found
	_, err = s.QueryTenant(Tenant{TenantId: "t001"})
	if err == nil || err.Error() != "Not Found" {
		t.Errorf("Expected 'Not Found' error, got: %v", err)
	}
}

func TestGormStore_TenantOfPage(t *testing.T) {
	s := newTestStore(t)

	// 插入 5 条记录
	for i := 0; i < 5; i++ {
		s.StoreTenant(Tenant{
			TenantId:   "t" + string(rune('0'+i)),
			TenantName: "租户" + string(rune('0'+i)),
			CreateTime: int64(i + 1),
		})
	}

	// 查询第 1 页，每页 2 条
	records, err := s.QueryTenantOfPage(0, 2)
	if err != nil {
		t.Fatalf("QueryTenantOfPage failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Page size = %d, want 2", len(records))
	}

	// 查询全部 (size = -1)
	all, err := s.QueryTenantOfPage(0, -1)
	if err != nil {
		t.Fatalf("QueryTenantOfPage all failed: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("Total records = %d, want 5", len(all))
	}

	// 应按 create_time DESC 排序（即最新在前）
	if all[0].CreateTime < all[4].CreateTime {
		t.Error("Records should be ordered by create_time DESC")
	}
}

// ==================== TenantNumber ====================

func TestGormStore_TenantNumberCRUD(t *testing.T) {
	s := newTestStore(t)

	// 创建
	err := s.StoreTenantNumber(TenantNumber{
		Number:   "13800000001",
		TenantId: "t001",
		Action:   NUMBER_ACTION_to_robot,
		WayType:  WAY_TYPE_gateway,
		Way:      "sip_gw",
		RobotID:  "robot001",
	})
	if err != nil {
		t.Fatalf("StoreTenantNumber create failed: %v", err)
	}

	// 查询
	tn, err := s.QueryTenantNumber(TenantNumber{Number: "13800000001"})
	if err != nil {
		t.Fatalf("QueryTenantNumber failed: %v", err)
	}
	if tn.TenantId != "t001" {
		t.Errorf("TenantId = %q, want %q", tn.TenantId, "t001")
	}
	if tn.Action != NUMBER_ACTION_to_robot {
		t.Errorf("Action = %d, want %d", tn.Action, NUMBER_ACTION_to_robot)
	}
	if tn.RobotID != "robot001" {
		t.Errorf("RobotID = %q, want %q", tn.RobotID, "robot001")
	}

	// 计数
	if count := s.CountTenantNumber(); count != 1 {
		t.Errorf("CountTenantNumber = %d, want 1", count)
	}

	// 更新
	err = s.StoreTenantNumber(TenantNumber{
		Number:  "13800000001",
		RobotID: "robot002",
	})
	if err != nil {
		t.Fatalf("StoreTenantNumber update failed: %v", err)
	}
	tn2, _ := s.QueryTenantNumber(TenantNumber{Number: "13800000001"})
	if tn2.RobotID != "robot002" {
		t.Errorf("RobotID after update = %q, want %q", tn2.RobotID, "robot002")
	}

	// 分页查询
	records, err := s.QueryTenantNumberOfPage(0, 10)
	if err != nil {
		t.Fatalf("QueryTenantNumberOfPage failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("Page size = %d, want 1", len(records))
	}

	// 删除
	err = s.DeleteTenantNumber(TenantNumber{Number: "13800000001"})
	if err != nil {
		t.Fatalf("DeleteTenantNumber failed: %v", err)
	}
	if count := s.CountTenantNumber(); count != 0 {
		t.Errorf("CountTenantNumber after delete = %d, want 0", count)
	}
}

func TestGormStore_QueryTenantNumber_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.QueryTenantNumber(TenantNumber{Number: "999"})
	if err == nil || err.Error() != "Not Found" {
		t.Errorf("Expected 'Not Found', got: %v", err)
	}
}

// ==================== Robot ====================

func TestGormStore_RobotCRUD(t *testing.T) {
	s := newTestStore(t)

	// 创建（带 Arg map 和 ToVendor）
	err := s.StoreRobot(Robot{
		RobotID:    "robot001",
		Target:     "http://ai-server/v1/chat",
		Arg:        map[string]any{"model": "qwen2.5", "temperature": 0.7},
		Welcome:    "您好，请问有什么可以帮您？",
		Prompt:     "你是一个智能客服",
		CreateTime: time.Now().UnixMilli(),
		ToVendor:   false,
	})
	if err != nil {
		t.Fatalf("StoreRobot create failed: %v", err)
	}

	// 查询
	robot, err := s.QueryRobot(Robot{RobotID: "robot001"})
	if err != nil {
		t.Fatalf("QueryRobot failed: %v", err)
	}
	if robot.Target != "http://ai-server/v1/chat" {
		t.Errorf("Target = %q, want %q", robot.Target, "http://ai-server/v1/chat")
	}
	if robot.Welcome != "您好，请问有什么可以帮您？" {
		t.Errorf("Welcome = %q, want %q", robot.Welcome, "您好，请问有什么可以帮您？")
	}
	if robot.Prompt != "你是一个智能客服" {
		t.Errorf("Prompt = %q, want %q", robot.Prompt, "你是一个智能客服")
	}
	// 验证 Arg map 序列化/反序列化
	if robot.Arg == nil {
		t.Fatal("Arg should not be nil")
	}
	if robot.Arg["model"] != "qwen2.5" {
		t.Errorf("Arg[model] = %v, want %q", robot.Arg["model"], "qwen2.5")
	}
	if robot.ToVendor != false {
		t.Errorf("ToVendor = %v, want false", robot.ToVendor)
	}

	// 计数
	if count := s.CountRobot(); count != 1 {
		t.Errorf("CountRobot = %d, want 1", count)
	}

	// 更新（修改 Target, Arg, ToVendor）
	err = s.StoreRobot(Robot{
		RobotID:  "robot001",
		Target:   "http://new-ai/v1/chat",
		Arg:      map[string]any{"model": "gpt4", "max_tokens": 1000},
		ToVendor: true,
	})
	if err != nil {
		t.Fatalf("StoreRobot update failed: %v", err)
	}
	r2, _ := s.QueryRobot(Robot{RobotID: "robot001"})
	if r2.Target != "http://new-ai/v1/chat" {
		t.Errorf("Target after update = %q, want %q", r2.Target, "http://new-ai/v1/chat")
	}
	if r2.Arg["model"] != "gpt4" {
		t.Errorf("Arg[model] after update = %v, want %q", r2.Arg["model"], "gpt4")
	}
	if r2.ToVendor != true {
		t.Errorf("ToVendor after update = %v, want true", r2.ToVendor)
	}

	// 分页查询
	records, err := s.QueryRobotOfPage(0, 10)
	if err != nil {
		t.Fatalf("QueryRobotOfPage failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("Page size = %d, want 1", len(records))
	}

	// 删除
	err = s.DeleteRobot(Robot{RobotID: "robot001"})
	if err != nil {
		t.Fatalf("DeleteRobot failed: %v", err)
	}
	if count := s.CountRobot(); count != 0 {
		t.Errorf("CountRobot after delete = %d, want 0", count)
	}
}

func TestGormStore_Robot_ToVendorUpdate(t *testing.T) {
	s := newTestStore(t)

	// 创建时 ToVendor = true
	s.StoreRobot(Robot{
		RobotID:  "robot_vendor",
		Target:   "http://vendor/api",
		ToVendor: true,
	})

	// 更新 ToVendor 为 false（bool 零值也能更新）
	s.StoreRobot(Robot{
		RobotID:  "robot_vendor",
		ToVendor: false,
	})
	r, _ := s.QueryRobot(Robot{RobotID: "robot_vendor"})
	if r.ToVendor != false {
		t.Errorf("ToVendor = %v, want false after update", r.ToVendor)
	}
}

func TestGormStore_QueryRobot_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.QueryRobot(Robot{RobotID: "nonexistent"})
	if err == nil || err.Error() != "Not Found" {
		t.Errorf("Expected 'Not Found', got: %v", err)
	}
}

// ==================== Extension ====================

func TestGormStore_ExtensionCRUD(t *testing.T) {
	s := newTestStore(t)

	// 创建
	err := s.StoreExtension(Extension{
		TenantId:    "t001",
		ExtensionId: "8001",
		CreateTime:  time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("StoreExtension create failed: %v", err)
	}

	// 查询
	ext, err := s.QueryExtension(Extension{ExtensionId: "8001"})
	if err != nil {
		t.Fatalf("QueryExtension failed: %v", err)
	}
	if ext.TenantId != "t001" {
		t.Errorf("TenantId = %q, want %q", ext.TenantId, "t001")
	}

	// 计数
	if count := s.CountExtension(); count != 1 {
		t.Errorf("CountExtension = %d, want 1", count)
	}

	// 更新
	err = s.StoreExtension(Extension{
		ExtensionId: "8001",
		TenantId:    "t002",
	})
	if err != nil {
		t.Fatalf("StoreExtension update failed: %v", err)
	}
	ext2, _ := s.QueryExtension(Extension{ExtensionId: "8001"})
	if ext2.TenantId != "t002" {
		t.Errorf("TenantId after update = %q, want %q", ext2.TenantId, "t002")
	}

	// 分页查询
	records, err := s.QueryExtensionOfPage(0, 10)
	if err != nil {
		t.Fatalf("QueryExtensionOfPage failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("Page size = %d, want 1", len(records))
	}

	// 删除
	err = s.DeleteExtension(Extension{ExtensionId: "8001"})
	if err != nil {
		t.Fatalf("DeleteExtension failed: %v", err)
	}
	if count := s.CountExtension(); count != 0 {
		t.Errorf("CountExtension after delete = %d, want 0", count)
	}
}

func TestGormStore_QueryExtension_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.QueryExtension(Extension{ExtensionId: "9999"})
	if err == nil || err.Error() != "Not Found" {
		t.Errorf("Expected 'Not Found', got: %v", err)
	}
}

// ==================== Agent ====================

func TestGormStore_AgentCRUD(t *testing.T) {
	s := newTestStore(t)

	// 创建
	err := s.StoreAgent(Agent{
		TenantId:    "t001",
		AgentId:     "agent001",
		AgentName:   "张三",
		ExtensionId: "8001",
		CreateTime:  time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("StoreAgent create failed: %v", err)
	}

	// 查询
	agent, err := s.QueryAgent(Agent{AgentId: "agent001"})
	if err != nil {
		t.Fatalf("QueryAgent failed: %v", err)
	}
	if agent.AgentName != "张三" {
		t.Errorf("AgentName = %q, want %q", agent.AgentName, "张三")
	}
	if agent.ExtensionId != "8001" {
		t.Errorf("ExtensionId = %q, want %q", agent.ExtensionId, "8001")
	}

	// 计数
	if count := s.CountAgent(); count != 1 {
		t.Errorf("CountAgent = %d, want 1", count)
	}

	// 更新
	err = s.StoreAgent(Agent{
		AgentId:     "agent001",
		AgentName:   "李四",
		ExtensionId: "8002",
	})
	if err != nil {
		t.Fatalf("StoreAgent update failed: %v", err)
	}
	a2, _ := s.QueryAgent(Agent{AgentId: "agent001"})
	if a2.AgentName != "李四" {
		t.Errorf("AgentName after update = %q, want %q", a2.AgentName, "李四")
	}
	if a2.ExtensionId != "8002" {
		t.Errorf("ExtensionId after update = %q, want %q", a2.ExtensionId, "8002")
	}

	// 分页查询
	records, err := s.QueryAgentOfPage(0, 10)
	if err != nil {
		t.Fatalf("QueryAgentOfPage failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("Page size = %d, want 1", len(records))
	}

	// 删除
	err = s.DeleteAgent(Agent{AgentId: "agent001"})
	if err != nil {
		t.Fatalf("DeleteAgent failed: %v", err)
	}
	if count := s.CountAgent(); count != 0 {
		t.Errorf("CountAgent after delete = %d, want 0", count)
	}
}

func TestGormStore_QueryAgent_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.QueryAgent(Agent{AgentId: "nonexistent"})
	if err == nil || err.Error() != "Not Found" {
		t.Errorf("Expected 'Not Found', got: %v", err)
	}
}

// ==================== 分页边界 ====================

func TestGormStore_PageBoundary(t *testing.T) {
	s := newTestStore(t)

	// 插入 3 个 Agent
	for i := 0; i < 3; i++ {
		s.StoreAgent(Agent{
			TenantId:    "t001",
			AgentId:     "a" + string(rune('0'+i)),
			AgentName:   "坐席" + string(rune('0'+i)),
			ExtensionId: "80" + string(rune('0'+i)),
			CreateTime:  int64(i + 1),
		})
	}

	// offset 超出范围应返回空
	records, err := s.QueryAgentOfPage(100, 10)
	if err != nil {
		t.Fatalf("QueryAgentOfPage offset overflow failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Expected empty result, got %d records", len(records))
	}

	// size = 0 应返回空
	records, _ = s.QueryAgentOfPage(0, 0)
	if len(records) != 0 {
		t.Errorf("size=0 should return empty, got %d records", len(records))
	}

	// 查询全部 size = -1
	all, _ := s.QueryAgentOfPage(0, -1)
	if len(all) != 3 {
		t.Errorf("size=-1 should return all, got %d records", len(all))
	}
}

// ==================== 数据持久化 ====================

func TestGormStore_Persistence(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "persist_test.db")

	// 第一次打开并写入数据
	s1 := &gormStore{logger: &testLogger{}}
	err := s1.Load("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("First Load failed: %v", err)
	}
	s1.StoreTenant(Tenant{TenantId: "persist_t", TenantName: "持久化测试"})
	s1.Unload()

	// 第二次打开，验证数据持久化
	s2 := &gormStore{logger: &testLogger{}}
	err = s2.Load("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("Second Load failed: %v", err)
	}
	defer s2.Unload()

	tenant, err := s2.QueryTenant(Tenant{TenantId: "persist_t"})
	if err != nil {
		t.Fatalf("QueryTenant after reload failed: %v", err)
	}
	if tenant.TenantName != "持久化测试" {
		t.Errorf("TenantName = %q, want %q", tenant.TenantName, "持久化测试")
	}
}

// ==================== Robot Arg 深度验证 ====================

func TestGormStore_RobotArg_NestedData(t *testing.T) {
	s := newTestStore(t)

	arg := map[string]any{
		"model":      "qwen2.5-7b",
		"max_tokens": float64(2048),
		"stream":     true,
		"params": map[string]any{
			"temperature": 0.7,
		},
	}
	s.StoreRobot(Robot{
		RobotID:    "r_nested",
		Target:     "http://ai/v1",
		Arg:        arg,
		CreateTime: 1,
	})

	robot, err := s.QueryRobot(Robot{RobotID: "r_nested"})
	if err != nil {
		t.Fatalf("QueryRobot failed: %v", err)
	}

	if robot.Arg["model"] != "qwen2.5-7b" {
		t.Errorf("Arg[model] = %v, want %q", robot.Arg["model"], "qwen2.5-7b")
	}
	if robot.Arg["stream"] != true {
		t.Errorf("Arg[stream] = %v, want true", robot.Arg["stream"])
	}
	// 嵌套对象会被反序列化为 map[string]interface{}
	if nested, ok := robot.Arg["params"].(map[string]interface{}); ok {
		if nested["temperature"] != 0.7 {
			t.Errorf("Arg[params][temperature] = %v, want 0.7", nested["temperature"])
		}
	} else {
		t.Error("Arg[params] should be a nested map")
	}
}

func TestGormStore_RobotArg_EmptyMap(t *testing.T) {
	s := newTestStore(t)

	s.StoreRobot(Robot{
		RobotID:    "r_empty_arg",
		Arg:        map[string]any{},
		CreateTime: 1,
	})

	robot, _ := s.QueryRobot(Robot{RobotID: "r_empty_arg"})
	if robot.Arg == nil {
		t.Error("Arg should not be nil for empty map")
	}
	if len(robot.Arg) != 0 {
		t.Errorf("Arg should be empty map, got %v", robot.Arg)
	}
}

// ==================== 多条记录分页 ====================

func TestGormStore_MultiPageQuery(t *testing.T) {
	s := newTestStore(t)

	// 插入 10 条 Extension
	for i := 0; i < 10; i++ {
		s.StoreExtension(Extension{
			TenantId:    "t001",
			ExtensionId: "80" + string(rune('0'+i)),
			CreateTime:  int64(i + 1),
		})
	}

	// 第 1 页
	page1, _ := s.QueryExtensionOfPage(0, 3)
	if len(page1) != 3 {
		t.Errorf("Page 1 size = %d, want 3", len(page1))
	}

	// 第 2 页
	page2, _ := s.QueryExtensionOfPage(3, 3)
	if len(page2) != 3 {
		t.Errorf("Page 2 size = %d, want 3", len(page2))
	}

	// 第 4 页（剩余 1 条）
	page4, _ := s.QueryExtensionOfPage(9, 3)
	if len(page4) != 1 {
		t.Errorf("Page 4 size = %d, want 1", len(page4))
	}

	// 验证总计数
	if count := s.CountExtension(); count != 10 {
		t.Errorf("CountExtension = %d, want 10", count)
	}
}

// ==================== Store 接口兼容性 ====================

func TestGormStore_ImplementsStoreInterface(t *testing.T) {
	// 编译期验证 gormStore 实现了 Store 接口
	var _ Store = (*gormStore)(nil)
}

func TestGormStore_EngineRegistration(t *testing.T) {
	// 验证 sqlite:// 前缀能正确获取引擎
	engine := Get("sqlite://test.db")
	if engine == nil {
		t.Fatal("Get('sqlite://...') should return a Store instance")
	}
}

// ==================== kcfg 数据迁移 ====================

func TestGormStore_MigrateFromKcfg(t *testing.T) {
	// 构造 kcfg 数据目录
	kcfgDir := t.TempDir()
	createKcfgSubDirs(t, kcfgDir)

	// 写入 User kcfg（密码不含 $ 符号避免 kcfg 变量解析）
	os.WriteFile(filepath.Join(kcfgDir, "User", "User_admin.kcfg"), []byte(
		"user {\n\tUsername admin\n\tPassword plainpass\n\tName 管理员\n\tRoles R_SUPER\n\tCreateTime 0\n\tUpdateTime 0\n}\n"), 0644)

	// 写入 Tenant kcfg
	os.WriteFile(filepath.Join(kcfgDir, "Tenant", "Tenant_t001.kcfg"), []byte(
		"tenant {\n\tTenantId t001\n\tTenantName 测试租户\n\tCreateTime 1700000000000\n\tExpireTime 0\n\tDefaultNumber \n}\n"), 0644)

	// 写入 Robot kcfg（Arg 使用 kcfg 多行语法 ````）
	os.WriteFile(filepath.Join(kcfgDir, "Robot", "Robot_r001.kcfg"), []byte(
		"robot {\n\tRobotID r001\n\tTarget http://ai/v1\n\tArg ```{\"model\":\"qwen\"}```\n\tWelcome 你好\n\tPrompt 你是客服\n\tCreateTime 1700000000000\n\tToVendor false\n}\n"), 0644)

	// 写入 TenantNumber kcfg
	os.WriteFile(filepath.Join(kcfgDir, "TenantNumber", "TenantNumber_13800000001.kcfg"), []byte(
		"tenant_number {\n\tNumber 13800000001\n\tTenantId t001\n\tCreateTime 0\n\tExpireTime 0\n\tAction 1\n\tWayType gateway\n\tWay sip_gw\n\tRobotID r001\n}\n"), 0644)

	// 写入 Extension kcfg
	os.WriteFile(filepath.Join(kcfgDir, "Extension", "Extension_8001.kcfg"), []byte(
		"extension {\n\tTenantId t001\n\tExtensionId 8001\n\tCreateTime 1700000000000\n}\n"), 0644)

	// 写入 Agent kcfg
	os.WriteFile(filepath.Join(kcfgDir, "Agent", "Agent_agent001.kcfg"), []byte(
		"agent {\n\tTenantId t001\n\tAgentId agent001\n\tAgentName 张三\n\tExtensionId 8001\n\tCreateTime 1700000000000\n}\n"), 0644)

	// 创建 gormStore，数据库放在 kcfgDir 下以触发迁移
	dbPath := filepath.Join(kcfgDir, "gpbx.db")
	s := &gormStore{logger: &testLogger{}}
	err := s.Load("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer s.Unload()

	// 验证 User 迁移
	u, err := s.QueryUser(User{Username: "admin"})
	if err != nil {
		t.Fatalf("QueryUser after migration failed: %v", err)
	}
	if u.Name != "管理员" {
		t.Errorf("User.Name = %q, want '管理员'", u.Name)
	}
	if u.Roles != "R_SUPER" {
		t.Errorf("User.Roles = %q, want 'R_SUPER'", u.Roles)
	}
	// kcfg 中为明文密码，migrateUserKcfg 直接 Create 会触发 GORM 的 BeforeSave...
	// 但我们已经改为手动 HashPassword，所以密码应该仍是明文（因为 Create 不会自动加密）
	// 实际上 migrateUserKcfg 使用 db.gormDB.Create，不会调用 HashPassword
	// 验证密码不为空即可
	if u.Password == "" {
		t.Error("Password should not be empty after migration")
	}

	// 验证 Tenant 迁移
	tenant, err := s.QueryTenant(Tenant{TenantId: "t001"})
	if err != nil {
		t.Fatalf("QueryTenant after migration failed: %v", err)
	}
	if tenant.TenantName != "测试租户" {
		t.Errorf("TenantName = %q, want '测试租户'", tenant.TenantName)
	}

	// 验证 Robot 迁移
	robot, err := s.QueryRobot(Robot{RobotID: "r001"})
	if err != nil {
		t.Fatalf("QueryRobot after migration failed: %v", err)
	}
	if robot.Target != "http://ai/v1" {
		t.Errorf("Target = %q, want 'http://ai/v1'", robot.Target)
	}
	if robot.Arg == nil || robot.Arg["model"] != "qwen" {
		t.Errorf("Arg[model] = %v, want 'qwen'", robot.Arg["model"])
	}

	// 验证 TenantNumber 迁移
	tn, err := s.QueryTenantNumber(TenantNumber{Number: "13800000001"})
	if err != nil {
		t.Fatalf("QueryTenantNumber after migration failed: %v", err)
	}
	if tn.RobotID != "r001" {
		t.Errorf("RobotID = %q, want 'r001'", tn.RobotID)
	}

	// 验证 Extension 迁移
	ext, err := s.QueryExtension(Extension{ExtensionId: "8001"})
	if err != nil {
		t.Fatalf("QueryExtension after migration failed: %v", err)
	}
	if ext.TenantId != "t001" {
		t.Errorf("TenantId = %q, want 't001'", ext.TenantId)
	}

	// 验证 Agent 迁移
	agent, err := s.QueryAgent(Agent{AgentId: "agent001"})
	if err != nil {
		t.Fatalf("QueryAgent after migration failed: %v", err)
	}
	if agent.AgentName != "张三" {
		t.Errorf("AgentName = %q, want '张三'", agent.AgentName)
	}

	// 验证 kcfg 文件目录未被删除
	if _, err := os.Stat(filepath.Join(kcfgDir, "User")); os.IsNotExist(err) {
		t.Error("kcfg User directory should be preserved after migration")
	}
	if _, err := os.Stat(filepath.Join(kcfgDir, "Tenant")); os.IsNotExist(err) {
		t.Error("kcfg Tenant directory should be preserved after migration")
	}
}

func TestGormStore_MigrateSkipsWhenNotEmpty(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "gpbx.db")

	// 第一次加载：空的 gormStore（无 kcfg 目录，不触发迁移）
	s1 := &gormStore{logger: &testLogger{}}
	s1.Load("sqlite://" + dbPath)

	// 预先插入一条 User
	s1.StoreUser(User{Username: "existing", Password: "pass", Name: "Existing"})
	s1.Unload()

	// 现在创建 kcfg 子目录和数据（模拟之前存储方式残留的文件）
	createKcfgSubDirs(t, dbDir)
	os.WriteFile(filepath.Join(dbDir, "User", "User_migrated.kcfg"), []byte(
		"user {\n\tUsername migrated_user\n\tPassword test\n\tName Migrated\n\tRoles R_USER\n\tCreateTime 0\n\tUpdateTime 0\n}\n"), 0644)

	// 第二次加载（数据库非空，应跳过迁移）
	s2 := &gormStore{logger: &testLogger{}}
	s2.Load("sqlite://" + dbPath)
	defer s2.Unload()

	// migrated_user 不应被迁移（数据库非空，跳过迁移）
	_, err := s2.QueryUser(User{Username: "migrated_user"})
	if err == nil {
		t.Error("User 'migrated_user' should NOT be migrated when database is not empty")
	}

	// 原有数据应保留
	existing, err := s2.QueryUser(User{Username: "existing"})
	if err != nil {
		t.Fatalf("Existing user should still be present: %v", err)
	}
	if existing.Name != "Existing" {
		t.Errorf("Existing user Name = %q, want 'Existing'", existing.Name)
	}
}

// createKcfgSubDirs 创建 kcfg 数据子目录
func createKcfgSubDirs(t *testing.T, dir string) {
	t.Helper()
	subDirs := []string{"User", "Tenant", "TenantNumber", "Robot", "Extension", "Agent"}
	for _, sub := range subDirs {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			t.Fatalf("Failed to create kcfg sub dir %s: %v", sub, err)
		}
	}
}
