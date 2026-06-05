package callcenter_api

import (
	"strings"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

/*
	{
		"task_id":"任务 ID",  // 外呼时，必须是 originate 指定的 task_id；呼入时由 task_create 中 task_id 指定
		"batch":[
			{
				"type":"file|tts|url",
				"content":""
			}
		],
		"is_break":true/false, // 是否打断前面一句正在说话内容
		"is_hangup":true/false, // 是否播放完就挂断通话
	}
*/

func api_playback(c *gin.Context) {
	var req_obj pbx.Playback
	c.ShouldBindBodyWithJSON(&req_obj)
	if len(req_obj.Batch) == 0 {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "batch missed or not array",
		})
		defaultLogger.Error(ThisModule, "api_playback batch missed or not array")
		return
	}
	task := datacenter.GetTaskManager().Get(req_obj.TaskID)
	uid := uuid.New()
	if task == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_playback not found task %v", req_obj.TaskID)
		return
	}
	switch task.TaskType {
	case datacenter.TASK_TYPE_callin_to_robot_vendor:
		fallthrough
	case datacenter.TASK_TYPE_originate_to_robot_vendor:
		{
			// 如果是呼出转机器人，需要使用 robot session 进行播放
			ret := pbx.RobotPlayback(uid.String(), task.B_Session.SessionID, req_obj)
			res_data, ok := ret.Data.(*event.CustomPromise)
			if !ok {
				res_data = &event.CustomPromise{}
			}
			if ret.Code == 0 {
				c.JSON(200, Response{
					Code: 200,
					Msg:  "Success",
				})
				defaultLogger.Info(ThisModule, "<%v> Playback Success", req_obj.TaskID)
			} else {
				c.JSON(200, Response{
					Code: 400,
					Msg:  res_data.Message,
				})
				defaultLogger.Info(ThisModule, "<%v> Playback Failed, Reason: <%v:%v>", req_obj.TaskID, ret.Code, ret.Data)
			}
		}
	default:
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Support",
		})
		defaultLogger.Error(ThisModule, "api_playback don't support %v task type", task.TaskType)
	}
}

type PlayAndGetDtmf struct {
	TaskID    string `json:"task_id"`
	SessionID string `json:"session_id"`
	File      string `json:"file"`
	HopeDigit string `json:"hope_digit"`
	Timeout   int    `json:"timeout"`
}

type PlayAndGetDtmfs struct {
	TaskID    string `json:"task_id"`
	SessionID string `json:"session_id"`
	File      string `json:"file"`
	HopeLen   int    `json:"hope_len"`
	Timeout   int    `json:"timeout"`
}

type Play struct {
	TaskID    string `json:"task_id"`
	SessionID string `json:"session_id"`
	File      string `json:"file"`
}

func api_play_and_getdigit(c *gin.Context) {
	var req_obj PlayAndGetDtmf
	_ = c.ShouldBindBodyWithJSON(&req_obj)
	task := datacenter.GetTaskManager().Get(req_obj.TaskID)
	if task == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_play_and_getdigit not found task_id %v", req_obj.TaskID)
		return
	}
	session := datacenter.GetSessionManager().Get(req_obj.SessionID)
	if session == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_play_and_getdigit not found session_id %v", req_obj.SessionID)
		return
	}

	if !task.Lock() {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Task Busy",
		})
		defaultLogger.Error(ThisModule, "Task:%v SessionID:%v api_play_and_getdigit Task Busy", req_obj.TaskID, req_obj.SessionID)
		return
	}
	defer task.Unlock()

	play_done := false
	go func() {
		pbx.SessionPlayback(req_obj.SessionID, req_obj.File, req_obj.Timeout/1000)
		play_done = true
	}()
	session.EnableDtmf(true)
	defer session.EnableDtmf(false)
	step := 50
	cur := 0

	for session.HangupTime <= 0 && cur < req_obj.Timeout {
		dtmf := session.GetDtmfs()
		if len(dtmf) > 0 {
			for _, d := range dtmf {
				if strings.Contains(req_obj.HopeDigit, string(d)) {
					if !play_done {
						pbx.ApiBreak(req_obj.SessionID)
					}
					c.JSON(200, Response{
						Code: 200,
						Msg:  "Success",
						Data: string(d),
					})
					defaultLogger.Info(ThisModule, "Task:%v SessionID:%v api_play_and_getdigit Success, dtmf: %v",
						req_obj.TaskID,
						req_obj.SessionID,
						string(d))
					return
				}
			}
		}

		if play_done {
			cur += step
		}
		time.Sleep(time.Millisecond * time.Duration(step))
	}
	c.JSON(200, Response{
		Code: 300,
		Msg:  "Failed",
	})
	defaultLogger.Info(ThisModule, "Task:%v SessionID:%v api_play_and_getdigit Failed",
		req_obj.TaskID,
		req_obj.SessionID)
}

func api_play_and_getdigits(c *gin.Context) {
	var req_obj PlayAndGetDtmfs
	_ = c.ShouldBindBodyWithJSON(&req_obj)
	task := datacenter.GetTaskManager().Get(req_obj.TaskID)
	if task == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_play_and_getdigits not found task_id %v", req_obj.TaskID)
		return
	}
	session := datacenter.GetSessionManager().Get(req_obj.SessionID)
	if session == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_play_and_getdigits not found session_id %v", req_obj.SessionID)
		return
	}

	if !task.Lock() {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Task Busy",
		})
		defaultLogger.Error(ThisModule, "Task:%v SessionID:%v api_play_and_getdigits Task Busy", req_obj.TaskID, req_obj.SessionID)
		return
	}
	defer task.Unlock()

	play_done := false
	go func() {
		pbx.SessionPlayback(req_obj.SessionID, req_obj.File, req_obj.Timeout/1000)
		play_done = true
	}()
	session.EnableDtmf(true)
	defer session.EnableDtmf(false)
	step := 50
	cur := 0
	dtmfs := ""
	for session.HangupTime <= 0 && cur < req_obj.Timeout {
		dtmfs += session.GetDtmfs()
		if len(dtmfs) >= req_obj.HopeLen {
			if !play_done {
				pbx.ApiBreak(req_obj.SessionID)
			}
			c.JSON(200, Response{
				Code: 200,
				Msg:  "Success",
				Data: dtmfs[:req_obj.HopeLen],
			})
			defaultLogger.Info(ThisModule, "Task:%v SessionID:%v api_play_and_getdigits Success, dtmf: %v",
				req_obj.TaskID,
				req_obj.SessionID,
				dtmfs[:req_obj.HopeLen])
			return
		}

		if play_done {
			cur += step
		}
		time.Sleep(time.Millisecond * time.Duration(step))
	}
	c.JSON(200, Response{
		Code: 300,
		Msg:  "Failed",
	})
	defaultLogger.Info(ThisModule, "Task:%v SessionID:%v api_play_and_getdigits Failed, dtmf: %v",
		req_obj.TaskID,
		req_obj.SessionID,
		dtmfs)
}

func api_play(c *gin.Context) {
	var req_obj Play
	_ = c.ShouldBindBodyWithJSON(&req_obj)
	task := datacenter.GetTaskManager().Get(req_obj.TaskID)
	if task == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_play_and_getdigit not found task_id %v", req_obj.TaskID)
		return
	}
	session := datacenter.GetSessionManager().Get(req_obj.SessionID)
	if session == nil {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Not Found Task",
		})
		defaultLogger.Error(ThisModule, "api_play_and_getdigit not found session_id %v", req_obj.SessionID)
		return
	}
	if !task.Lock() {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Task Busy",
		})
		defaultLogger.Error(ThisModule, "Task:%v SessionID:%v api_play Task Busy", req_obj.TaskID, req_obj.SessionID)
		return
	}
	defer task.Unlock()
	result := pbx.SessionPlayback(req_obj.SessionID, req_obj.File, 120)
	if result.Code != 0 {
		c.JSON(200, Response{
			Code: 400,
			Msg:  "Play Failed",
		})
	} else {
		c.JSON(200, Response{
			Code: 200,
			Msg:  "Success",
		})
	}
	defaultLogger.Info(ThisModule, "Task:%v SessionID:%v api_play result:%v", req_obj.TaskID, req_obj.SessionID, result)
}
func init() {
	routers["/api/playback"] = api_playback
	routers["/api/play_and_getdigit"] = api_play_and_getdigit
	routers["/api/play_and_getdigits"] = api_play_and_getdigits
	routers["/api/play"] = api_play
}
