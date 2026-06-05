package callcenter_api

import (
	// "gitee.com/kolonse_zhjsh/gpbx/web"

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
		"type":"to_robot|to_ivr|to_user|to_acd|park",
		"tenant_id":"",   //企业ID
		"display_number":"",  外显号码
		"number":"",  // 呼叫号码
		"originate_type":"gateway|user|ims", // 呼出方式 网关方式(sofia/gateway/xxx/number)，本地方式(user/1001)，中继方式(sofia/exteranl/number@ip:port)
		"originate_arg":"gateway name|ip:port" , originate_type 是 gateway/ims时必填
		"calbback":"http://192.168.247.128:8080/robot/asr/notify",
		"to_robot":{
			"target":"robot/asr_xxx/tts_xxx/ai_xxx",  type 为 to_robot时有效
			"arg":{
				"asr":{    //  asr 参数， 提供给asr 模块使用

				}，
				"tts":{    // tts 参数, 提供给 tts 模块使用
				},
				"ai":{    // ai 参数，提供给 ai 智能体使用
				}
			}
		},
		"to_ua":{
			"target":"坐席分机"
		},
		"ivr_id": "10000"
	}
*/

// type Response struct {
// 	Code    int    `json:"code"`
// 	Message string `json:"message"`
// }

func originate_to_ivr(req_obj pbx.Originate, c *gin.Context) {
	err := pbx.OriginateToIvr(req_obj)
	if err != nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  err.Error(),
		})
		defaultLogger.Info(ThisModule, "<%v> Originate %v %v Failed, reason:%v", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number, err.Error())
		return
	}
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
	defaultLogger.Info(ThisModule, "<%v> Originate %v %v Success", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number)
}

func originate_to_robot(req_obj pbx.Originate, c *gin.Context) {
	uid := uuid.New()
	task := datacenter.GetTaskManager().CreateAndGet(req_obj.TaskID)
	task.Lock()
	defer task.Unlock()

	task.TenantId = req_obj.TenantID
	task.A_Session.TaskID = req_obj.TaskID
	task.A_Session.SessionID = req_obj.TaskID
	task.A_Session.Caller = req_obj.DisplayNumber
	task.A_Session.Callee = req_obj.Number
	task.ToRobotID = req_obj.RobotID
	if len(task.ToRobotID) == 0 {
		task.TaskType = datacenter.TASK_TYPE_originate_to_robot_vendor
		task.ToRobotID = req_obj.ToRobot.GetRobotID()
		_, err := app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{
			RobotID: task.ToRobotID,
		})
		if err != nil {
			_ = app.GetDefaultApp().GetStoreEngine().StoreRobot(store.Robot{
				RobotID:    task.ToRobotID,
				Target:     req_obj.ToRobot.Target,
				Arg:        req_obj.ToRobot.Arg,
				CreateTime: time.Now().UnixMicro(),
				ToVendor:   true,
			})
		}
	} else {
		robot, err := app.GetDefaultApp().GetStoreEngine().QueryRobot(store.Robot{
			RobotID: task.ToRobotID,
		})
		if err != nil {
			c.JSON(200, Response{
				Code: 400,
				Msg:  "robot not found",
			})
			defaultLogger.Error(ThisModule, "originate_to_robot robot not found")
			return
		}
		if robot.ToVendor {
			task.TaskType = datacenter.TASK_TYPE_originate_to_robot_vendor
		} else {
			task.TaskType = datacenter.TASK_TYPE_originate_to_robot_ai
		}
	}
	task.AsrCallback = req_obj.AsrCallback
	datacenter.GetSessionManager().Set(task.A_Session.SessionID, task.A_Session)
	ret := pbx.OriginateAndPark(uid.String(),
		req_obj.TaskID,
		req_obj.DisplayNumber,
		req_obj.Number,
		req_obj.OriginateType,
		req_obj.OriginateArg,
		req_obj.Timeout)
	res_data, ok := ret.Data.(*event.CustomPromise)
	if !ok {
		res_data = &event.CustomPromise{}
	}
	if ret.Code == 0 {
		c.JSON(200, Response{
			Code: 200,
			Msg:  "Success",
		})
		defaultLogger.Info(ThisModule, "<%v> Originate %v %v Success", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number)
	} else {
		c.JSON(200, Response{
			Code: 400,
			Msg:  res_data.Message,
		})
		defaultLogger.Info(ThisModule, "<%v> Originate %v %v Failed, reason:<%v:%v>", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number, ret.Code, ret.Data)
	}
}

func api_originate(c *gin.Context) {
	var req_obj pbx.Originate
	_ = c.ShouldBindBodyWithJSON(&req_obj)
	if len(req_obj.Type) == 0 {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "type missed or not string",
		})
		defaultLogger.Error(ThisModule, "api_originate type missed or not string")
		return
	}
	switch req_obj.Type {
	case "to_ivr":
		originate_to_ivr(req_obj, c)
	case "to_robot":
		originate_to_robot(req_obj, c)
	default:
		c.JSON(200, Response{
			Code: 400,
			Msg:  "not support type : " + req_obj.Type,
		})
		defaultLogger.Error(ThisModule, "api_originate not support type : %v", req_obj.Type)
	}
}

func init() {
	routers["/api/originate"] = api_originate
}
