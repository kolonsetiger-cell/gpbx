package pbx

import (
	"encoding/base64"
	"encoding/json"

	"gitee.com/kolonse_zhjsh/gpbx/event"
)

type Transfer struct {
	TaskID    string `json:"task_id"`
	SessionID string `json:"session_id"`
	Type      string `json:"type"`
	// DisplayNumber string `json:"display_number"`
	// Number        string `json:"number"`
	ToRobot     Param  `json:"to_robot"`
	RobotID     string `json:"robot_id"`
	AsrCallback string `json:"asr_callback"`
}

func TransferToRobot(job_uuid string, task_id string, session_id string, display_number, number string, robot_session_id string, robot_arg map[string]any) event.Result {
	fix_arg(&job_uuid)
	fix_arg(&task_id)
	fix_arg(&session_id)
	fix_arg(&display_number)
	fix_arg(&number)
	fix_arg(&robot_session_id)

	arg, _ := json.Marshal(robot_arg)
	cmd := event.PBXCommand{
		JobId: job_uuid,
		Cmd:   "luarun",
		Arg: format_arg("do_transfer_to_robot.lua",
			job_uuid,
			task_id,
			session_id,
			display_number,
			number,
			robot_session_id,
			base64.StdEncoding.EncodeToString(arg)),
		Uuid:    job_uuid,
		Timeout: 10,
	}

	ret := event.GetDefaultBus().Request(event.TOPIC_SEND_API_WITH_PROMISE, cmd)
	if ret.Code == 0 {
		if ret.Data == nil {
			ret.Code = ERROR_CODE_TIMEOUT // 表示超时
		} else {
			rsp, ok := ret.Data.(*event.CustomPromise)
			if !ok {
				ret.Code = ERROR_CODE_INTERNAL_ERROR // 表示接收数据格式错误
			} else {
				ret.Code = rsp.Code
			}
		}
	}
	return ret
}
