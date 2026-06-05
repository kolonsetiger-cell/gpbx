package router

import (
	"strconv"

	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/google/uuid"
)

func (n *Router) to_robot(task *datacenter.Task, robot store.Robot) {
	uid := uuid.New()
	datacenter.GetSessionManager().Set(uid.String(), task.B_Session)
	task.B_Session.TaskID = task.TaskID
	task.B_Session.SessionID = uid.String()
	task.B_Session.Caller = task.A_Session.Caller
	task.B_Session.Callee = robot.Target
	task.B_Session.IsRobot = true
	task.B_Session.RobotID = robot.RobotID

	ret := pbx.TransferToRobot(uid.String(),
		task.TaskID,
		task.A_Session.SessionID,
		task.A_Session.Caller,
		task.B_Session.Callee,
		uid.String(),
		robot.Arg)
	res_data, ok := ret.Data.(*event.CustomPromise)
	if !ok {
		res_data = &event.CustomPromise{}
	}
	if ret.Code != 0 {
		pbx.HangupCall(task.A_Session.SessionID)
	} else {
		tm, _ := strconv.Atoi(res_data.Time)
		task.A_Session.BridgeTime = int64(tm)
		task.B_Session.BridgeTime = int64(tm)
		gsession := datacenter.GetGlobalSessionManager().Get(task.B_Session.SessionID)
		if gsession != nil {
			task.B_Session.CallTime = gsession.CallTime
			task.B_Session.AnswerTime = gsession.AnswerTime
		}
	}
	defaultLogger.Info(ThisModule, "<%v> To Robot Completed, Result: <%v:%v>", task.TaskID, ret.Code, ret.Data)
}
