package callcenter_api

import (
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

/*
*
	{
		"task_id":"任务 ID",
		"session_id":"", // 要转接的 session_id， 另一路 session_id 会挂断
		"type":"to_robot|to_ivr|to_user|to_acd",
		"asr_calbback":"http://192.168.247.128:8080/robot/asr/notify", // to_robot 并且有 asr时，可以自定义 ASR 回调地址
		"to_user":{ // type 为 to_user 时有效
			"display_number":"",  外显号码
			"number":"",  // 呼叫号码
			"originate_type":"gateway|user|ims", // 呼出方式 网关方式(sofia/gateway/xxx/number)，本地方式(user/1001)，中继方式(sofia/exteranl/number@ip:port)
			"originate_arg":"gateway name|ip:port" , originate_type 是 gateway/ims时必填
			"timeout":10,
		}
		"to_robot":{ // type 为 to_robot 时有效
			"target":"robot/asr_xxx/tts_xxx/ai_xxx",  type 为 to_robot时有效
			"arg":{
				"asr":{    //  asr 参数， 提供给asr 模块使用
					"calbback":"http://192.168.247.128:8080/robot/asr/notify"
				}，
				"tts":{    // tts 参数, 提供给 tts 模块使用
				},
				"ai":{    // ai 参数，提供给 ai 智能体使用
				}
			}
		}
	}
*/

func transfer_to_robot(req_obj pbx.Transfer, task *datacenter.Task, c *gin.Context) {
	uid := uuid.New()
	if !task.Lock() {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "task is locked",
		})
		defaultLogger.Error(ThisModule, "transfer_to_robot task is locked")
		return
	}
	defer task.Unlock()
	task.ToRobotID = req_obj.RobotID
	if len(task.ToRobotID) == 0 {
		task.ToRobotID = req_obj.ToRobot.GetRobotID()
		_, err := app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{
			RobotID: task.ToRobotID,
		})
		if err != nil {
			app.GetDefaultApp().GetStoreEngine().StoreRobot(store.Robot{
				RobotID:    task.ToRobotID,
				Target:     req_obj.ToRobot.Target,
				Arg:        req_obj.ToRobot.Arg,
				CreateTime: time.Now().UnixMicro(),
				ToVendor:   true,
			})
		}
	}
	robot, err := app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{
		RobotID: task.ToRobotID,
	})
	if err != nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "robot not found",
		})
		defaultLogger.Error(ThisModule, "transfer_to_robot robot not found")
		return
	}
	task.AsrCallback = req_obj.AsrCallback
	// task.Transfer(req_obj.SessionID, uid.String())
	new_session := task.NewSession(uid.String())
	// task.Transfer(req_obj.SessionID, new_session)
	datacenter.GetSessionManager().Set(uid.String(), new_session)
	new_session.TaskID = task.TaskID
	new_session.Caller = task.A_Session.Caller
	new_session.Callee = robot.Target
	new_session.IsRobot = true
	new_session.RobotID = robot.RobotID

	ret := pbx.TransferToRobot(uid.String(),
		task.TaskID,
		req_obj.SessionID,
		task.A_Session.Caller,
		task.B_Session.Callee,
		uid.String(),
		robot.Arg)
	res_data, ok := ret.Data.(*event.CustomPromise)
	if !ok {
		res_data = &event.CustomPromise{}
	}
	if ret.Code != 0 {
		// pbx.HangupCall(task.A_Session.SessionID)
		c.JSON(200, Response{
			Code: 400,
			Msg:  res_data.Message,
		})
		defaultLogger.Error(ThisModule, "<%v> Transfer %v %v Failed, reason:<%v:%v>", req_obj.TaskID, req_obj.SessionID, robot.Target, ret.Code, ret.Data)
	} else {
		tm, _ := strconv.Atoi(res_data.Time)
		task.Transfer(req_obj.SessionID, new_session)
		if robot.ToVendor {
			task.TaskType = datacenter.TASK_TYPE_transfer_to_robot_vendor
		} else {
			task.TaskType = datacenter.TASK_TYPE_transfer_to_robot_ai
		}
		task.A_Session.BridgeTime = int64(tm)
		task.B_Session.BridgeTime = int64(tm)
		gsession := datacenter.GetGlobalSessionManager().Get(task.B_Session.SessionID)
		if gsession != nil {
			task.B_Session.CallTime = gsession.CallTime
			task.B_Session.AnswerTime = gsession.AnswerTime
		}
	}
}

func api_transfer(c *gin.Context) {
	var req_obj pbx.Transfer
	err := c.ShouldBindBodyWithJSON(&req_obj)
	if err != nil {
		panic(err)
	}
	task := datacenter.GetTaskManager().Get(req_obj.TaskID)
	if task == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_transfer not found task %v", req_obj.TaskID)
		return
	}
	switch req_obj.Type {
	case "to_robot":
		transfer_to_robot(req_obj, task, c)
		return
	default:
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Support",
		})
		defaultLogger.Error(ThisModule, "api_transfer don't support %v type", req_obj.Type)
		return
	}
}

func init() {
	routers["/api/transfer"] = api_transfer
}
