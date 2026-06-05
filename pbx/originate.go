package pbx

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/google/uuid"
)

const (
	ERROR_CODE_TIMEOUT        = -100
	ERROR_CODE_INTERNAL_ERROR = -101
	ERROR_CODE_PARAM_ERROR    = -102
)

func fix_arg(val *string) {
	if len(*val) == 0 {
		*val = "-"
	}
}

func format_arg(arg ...string) string {
	var res strings.Builder
	for i, v := range arg {
		res.WriteString(v)
		if i != len(arg)-1 {
			res.WriteString(" ")
		}
	}
	return res.String()
}

type Param struct {
	Target string         `json:"target"`
	Arg    map[string]any `json:"arg"`
}

func (p *Param) GetRobotID() string {
	arg, _ := json.Marshal(p.Arg)
	hash := md5.New()
	hash.Write([]byte(p.Target + string(arg)))
	return hex.EncodeToString(hash.Sum(nil))
}

type Originate struct {
	TaskID        string `json:"task_id"`
	Type          string `json:"type"`
	TenantID      string `json:"tenant_id"`
	DisplayNumber string `json:"display_number"`
	Number        string `json:"number"`
	OriginateType string `json:"originate_type"`
	OriginateArg  string `json:"originate_arg"`
	Timeout       int    `json:"timeout"`
	ToRobot       Param  `json:"to_robot"`
	RobotID       string `json:"robot_id"`
	AsrCallback   string `json:"asr_callback"`
	IvrID         string `json:"ivr_id"`
}

func OriginateToIvr(req_obj Originate) error {
	if len(req_obj.IvrID) == 0 {
		return fmt.Errorf("ivr_id is empty")
	}

	// ivr, err := app.GetDefaultApp().GetStoreEngine().QueryIvr(store.Ivr{
	// 	IvrID: req_obj.IvrID,
	// })

	// if err != nil || ivr.IvrID == "" {
	// 	return fmt.Errorf("ivr_id not found")
	// }
	var tenantNumber store.TenantNumber
	if req_obj.OriginateType == "" {
		tn, err := app.GetDefaultApp().GetStoreEngine().QueryTenantNumber(store.TenantNumber{
			TenantId: req_obj.TenantID,
			Number:   req_obj.DisplayNumber,
		})
		if err != nil || tn.TenantId == "" {
			return fmt.Errorf("tenant_id not found")
		}
		tenantNumber = tn
	} else {
		tenantNumber.WayType = req_obj.OriginateType
		tenantNumber.Way = req_obj.OriginateArg
	}

	uid := uuid.New()
	if req_obj.TaskID == "" {
		req_obj.TaskID = uid.String()
	}
	task := datacenter.GetTaskManager().CreateAndGet(req_obj.TaskID)
	task.Lock()
	defer task.Unlock()

	task.TenantId = req_obj.TenantID
	task.A_Session.TaskID = req_obj.TaskID
	task.A_Session.SessionID = req_obj.TaskID
	task.A_Session.Caller = req_obj.DisplayNumber
	task.A_Session.Callee = req_obj.Number
	task.ToIvrID = req_obj.IvrID
	task.TaskType = datacenter.TASK_TYPE_originate_to_ivr
	datacenter.GetSessionManager().Set(task.A_Session.SessionID, task.A_Session)
	ret := OriginateAndPark(uid.String(), req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number, tenantNumber.WayType, tenantNumber.Way, req_obj.Timeout)
	if ret.Code == 0 {
		return nil
	} else {
		return fmt.Errorf("originate to ivr failed, reason:<%v:%v>", ret.Code, ret.Data)
	}
}

func OriginateAndPark(job_uuid, task_id, display_number, number, originate_type, originate_arg string, timeout int) event.Result {
	fix_arg(&job_uuid)
	fix_arg(&task_id)
	fix_arg(&display_number)
	fix_arg(&number)
	fix_arg(&originate_type)
	fix_arg(&originate_arg)

	if timeout <= 0 || timeout > 120 {
		timeout = 30
	}
	cmd := event.PBXCommand{
		JobId:   job_uuid,
		Cmd:     "luarun",
		Arg:     format_arg("do_originate_and_park.lua", job_uuid, task_id, display_number, number, originate_type, originate_arg, strconv.Itoa(timeout)),
		Uuid:    job_uuid,
		Timeout: timeout + 5,
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
				ret.Data = rsp.Message
			}
		}
	}
	return ret
}

func OriginateToRobot(job_uuid, task_id, display_number, number, originate_type, originate_arg string, timeout int, robot_id, robot_number string, robot_arg map[string]any) event.Result {
	fix_arg(&job_uuid)
	fix_arg(&task_id)
	fix_arg(&display_number)
	fix_arg(&number)
	fix_arg(&originate_type)
	fix_arg(&originate_arg)
	fix_arg(&robot_id)
	fix_arg(&robot_number)

	if timeout <= 0 || timeout > 120 {
		timeout = 30
	}
	arg, _ := json.Marshal(robot_arg)
	cmd := event.PBXCommand{
		JobId: job_uuid,
		Cmd:   "luarun",
		Arg: format_arg("do_originate_to_robot.lua",
			job_uuid,
			task_id,
			display_number,
			number,
			originate_type,
			originate_arg,
			strconv.Itoa(timeout),
			robot_id,
			robot_number,
			base64.StdEncoding.EncodeToString(arg)),
		Uuid:    job_uuid,
		Timeout: timeout + 5,
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
