package datacenter

import (
	"sync"
	"time"
)

const (
	TASK_TYPE_originate                 = 0x10000 // 呼出类型
	TASK_TYPE_originate_to_robot        = 0x10001 // 呼出转机器人
	TASK_TYPE_originate_to_robot_vendor = 0x10011 // 呼出转机器人，转三方
	TASK_TYPE_originate_to_robot_ai     = 0x10021 // 呼出转机器人，转自建AI
	TASK_TYPE_originate_to_ivr          = 0x10002 // 呼出转 IVR
	TASK_TYPE_originate_to_user         = 0x10003 // 呼出转用户
	TASK_TYPE_originate_to_acd          = 0x10004 // 呼出转排队
	TASK_TYPE_originate_to_park         = 0x10005 // 仅仅呼出
	TASK_TYPE_callin                    = 0x20000 // 表示呼入
	TASK_TYPE_callin_to_robot           = 0x20001 // 表示呼入后转robot
	TASK_TYPE_callin_to_robot_vendor    = 0x20011 // 表示呼入后转robot
	TASK_TYPE_callin_to_robot_ai        = 0x20021 // 表示呼入后转robot
	TASK_TYPE_transfer                  = 0x30000 // 转移
	TASK_TYPE_transfer_to_robot         = 0x30001 // 转移转机器人
	TASK_TYPE_transfer_to_robot_vendor  = 0x30011 // 转移转三方
	TASK_TYPE_transfer_to_robot_ai      = 0x30021 // 转移转自建AI
	TASK_TYPE_makecall                  = 0x40000 // 双呼
)

type Task struct {
	TaskID      string
	A_Session   *Session
	B_Session   *Session
	TaskType    int
	TenantId    string
	AsrCallback string
	CreateTime  int64
	ToRobotID   string
	lock        sync.Mutex
	lock_count  int
	ToIvrID     string
}

func (t *Task) IsLocked() bool {
	return t.lock_count > 0
}

func (t *Task) Lock() bool {
	if !t.lock.TryLock() {
		return false
	}
	t.lock_count++
	return true
}

func (t *Task) Unlock() {
	t.lock_count--
	t.lock.Unlock()
}

// func (t *Task) IsCallBridged() bool {
// 	return t.A_Session.AnswerTime > 0 &&
// 		t.B_Session.AnswerTime > 0 &&
// 		t.A_Session.BridgeTime > 0 &&
// 		t.B_Session.BridgeTime > 0
// }

func (t *Task) isCallCompleted() bool {
	if !t.A_Session.isValidCall() && !t.B_Session.isValidCall() {
		return true
	}

	return false
}

func (t *Task) Transfer(session_src string, session_to *Session) {
	if session_src == t.A_Session.SessionID {
		t.B_Session = session_to
	} else {
		t.A_Session = t.B_Session
		t.B_Session = session_to
	}
}

func (t *Task) NewSession(session_id string) *Session {
	return &Session{
		SessionID:  session_id,
		createTime: time.Now().UnixMilli(),
	}
}

type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func (t *TaskManager) CreateAndGet(id string) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	task := &Task{
		TaskID: id,
		A_Session: &Session{
			TaskID:     id,
			createTime: time.Now().UnixMilli(),
		},
		B_Session: &Session{
			TaskID:     id,
			createTime: time.Now().UnixMilli(),
		},
		CreateTime: time.Now().UnixMilli(),
		lock_count: 0,
	}
	t.tasks[id] = task
	return task
}

func (t *TaskManager) Get(id string) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	task, ok := t.tasks[id]
	if !ok {
		return nil
	}
	return task
}

func (t *TaskManager) Delete(id string) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	task, ok := t.tasks[id]
	if !ok {
		return nil
	}
	delete(t.tasks, id)
	return task
}

func (t *TaskManager) RemoveCompletedCall() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().UnixMilli()
	for k, v := range t.tasks {
		if v.isCallCompleted() && now-v.CreateTime > 2*60*1000 {
			delete(t.tasks, k)
		}
	}
}

var taskManager *TaskManager

func GetTaskManager() *TaskManager {
	return taskManager
}

func init() {
	taskManager = &TaskManager{
		tasks: make(map[string]*Task),
	}
}
