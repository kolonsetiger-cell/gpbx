package pbx

import (
	"strconv"

	"gitee.com/kolonse_zhjsh/gpbx/event"
)

type MakecallArg struct {
	JobUUID     string
	TaskID      string
	A_SessionID string
	A_Caller    string
	A_Callee    string
	A_WayType   string
	A_Way       string
	B_SessionID string
	B_Caller    string
	B_Callee    string
	B_WayType   string
	B_Way       string
	Timeout     int
}

func (p *MakecallArg) Fix() {
	fix_arg(&p.JobUUID)
	fix_arg(&p.TaskID)
	fix_arg(&p.A_SessionID)
	fix_arg(&p.A_Caller)
	fix_arg(&p.A_Callee)
	fix_arg(&p.A_WayType)
	fix_arg(&p.A_Way)
	fix_arg(&p.B_SessionID)
	fix_arg(&p.B_Caller)
	fix_arg(&p.B_Callee)
	fix_arg(&p.B_WayType)
	fix_arg(&p.B_Way)

	if p.Timeout <= 0 || p.Timeout > 120 {
		p.Timeout = 60
	}
}

func Makecall(arg MakecallArg) event.Result {
	arg.Fix()
	cmd := event.PBXCommand{
		JobId: arg.JobUUID,
		Cmd:   "luarun",
		Arg: format_arg("do_makecall.lua",
			arg.JobUUID,
			arg.TaskID,
			arg.A_SessionID,
			arg.A_Caller,
			arg.A_Callee,
			arg.A_WayType,
			arg.A_Way,
			arg.B_SessionID,
			arg.B_Caller,
			arg.B_Callee,
			arg.B_WayType,
			arg.B_Way,
			strconv.Itoa(arg.Timeout/2)),
		Uuid:    arg.JobUUID,
		Timeout: arg.Timeout,
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
