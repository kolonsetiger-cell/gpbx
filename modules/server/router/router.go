package router

import (
	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
)

const ThisModule = "Router"

var defaultLogger log.Logger

type Router struct {
	exit_sig chan bool
}

func (n *Router) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *Router) Init() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	return nil
}

func (n *Router) route_inbound(msg *event.SessionAnswer) {
	gsession := datacenter.GetGlobalSessionManager().Get(msg.SessionId)
	if gsession == nil {
		defaultLogger.Error(ThisModule, "Session:%v Not Found", msg.SessionId)
		return
	}
	if gsession.IsRobot {
		return
	}

	// 需要查询该被叫配置的租户
	number, err := app.GetDefaultApp().GetStoreEngine().QueryTenantNumber(store.TenantNumber{Number: msg.Callee})
	if err != nil {
		defaultLogger.Error(ThisModule, "Session:%v Caller:%v Callee:%v Not Found Poliy", msg.SessionId, msg.Caller, msg.Callee)
		// 需要挂断通话
		pbx.HangupCall(msg.SessionId)
		return
	}

	task := datacenter.GetTaskManager().CreateAndGet(msg.SessionId)
	task.Lock()
	defer task.Unlock()

	datacenter.GetSessionManager().Set(msg.SessionId, task.A_Session)
	task.A_Session.Caller = msg.Caller
	task.A_Session.Callee = msg.Callee
	task.A_Session.TaskID = msg.SessionId
	task.A_Session.SessionID = msg.SessionId

	task.A_Session.CallTime = gsession.CallTime
	task.A_Session.AnswerTime = gsession.AnswerTime
	task.TaskType = datacenter.TASK_TYPE_callin_to_robot
	robot, err := app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{RobotID: number.RobotID})
	if err != nil {
		defaultLogger.Error(ThisModule, "<%v> Session:%v Caller:%v Callee:%v Not Found RobotId %v", task.TaskID, msg.SessionId, msg.Caller, msg.Callee, number.RobotID)
		pbx.HangupCall(msg.SessionId)
		return
	}
	if robot.ToVendor {
		task.TaskType = datacenter.TASK_TYPE_callin_to_robot_vendor
	} else {
		task.TaskType = datacenter.TASK_TYPE_callin_to_robot_ai
	}
	defaultLogger.Info(ThisModule, "Session:%v Caller:%v Callee:%v Tenant %v To Action %v", msg.SessionId, msg.Caller, msg.Callee, number.TenantId, number.Action)
	switch number.Action {
	case store.NUMBER_ACTION_to_robot:
		n.to_robot(task, robot)
	case store.NUMBER_ACTION_to_vendor:
		// n.to_vendor(task, msg, number.VendorID)
	default:
		defaultLogger.Error(ThisModule, "Session:%v Caller:%v Callee:%v Not Support Action %v", msg.SessionId, msg.Caller, msg.Callee, number.Action)
		pbx.HangupCall(msg.SessionId)
		return
	}
}

func (n *Router) route_outbound(msg *event.SessionAnswer) {
	// session := datacenter.GetSessionManager().Get(msg.SessionId)
	// // 如果内存中有该 session 数据，说明不是呼入
	// if session != nil {
	// 	defaultLogger.Info(ThisModule, "Ignore Session:%v Answer", msg)
	// 	return
	// }
	gsession := datacenter.GetGlobalSessionManager().Get(msg.SessionId)
	if gsession == nil {
		defaultLogger.Error(ThisModule, "Session:%v Not Found", msg.SessionId)
		return
	}
	if gsession.IsRobot {
		return
	}
	task := datacenter.GetTaskManager().Get(msg.TaskId)
	if task == nil {
		defaultLogger.Error(ThisModule, "Task:%v Not Found", msg.TaskId)
		return
	}

	switch task.TaskType {
	case datacenter.TASK_TYPE_originate_to_robot_vendor:
		fallthrough
	case datacenter.TASK_TYPE_originate_to_robot_ai:
		robot, err := app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{RobotID: task.ToRobotID})
		if err != nil {
			defaultLogger.Error(ThisModule, "<%v> Session:%v Caller:%v Callee:%v Not Found RobotId %v", task.TaskID, msg.SessionId, msg.Caller, msg.Callee, task.ToRobotID)
			pbx.HangupCall(msg.SessionId)
			return
		}
		n.to_robot(task, robot)
		return
	default:
		return
	}
}

func (n *Router) Run() error {
	defaultLogger.Info(ThisModule, "Run Router")
	sess_answer_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_ANSWER)
	go func() {
		exit := false
		for !exit {
			select {
			case msg := <-sess_answer_chan:
				{
					answer_msg, ok := msg.Data.(*event.SessionAnswer)
					if !ok {
						defaultLogger.Error(ThisModule, "%v Answer Not Found", msg.Data)
						break
					}
					defaultLogger.Debug(ThisModule, "%v Answer", answer_msg)
					if answer_msg.Direction != "inbound" {
						n.route_outbound(answer_msg)
					} else {
						n.route_inbound(answer_msg)
					}
				}
			case <-n.exit_sig:
				exit = true
			}
		}
	}()
	return nil
}

func (n *Router) Uninit() error {
	defaultLogger.Info(ThisModule, "Uninit Router")
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	return nil
}

var defaultServer *Router

func init() {
	defaultServer = &Router{}
	app.GetDefaultApp().Add(0, defaultServer)
}
