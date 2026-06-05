package pbx

import (
	"encoding/base64"
	"strconv"
	"strings"

	"gitee.com/kolonse_zhjsh/gpbx/event"
)

type Content struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Playback struct {
	TaskID    string `json:"task_id"`
	Batch     []Content
	IsBreak   bool `json:"is_break"`
	IsHangup  bool `json:"is_hangup"`
	IsAsync   bool `json:"is_async"`
	LoopCount int  `json:"loop_count"`
}

const (
	CONTENT_TYPE_FILE = "file"
	CONTENT_TYPE_URL  = "url"
	CONTENT_TYPE_TTS  = "tts"
)

func RobotPlayback(job_uuid string, session_id string, play Playback) event.Result {
	timeout := 10
	for _, v := range play.Batch {
		switch v.Type {
		case "tts":
			timeout += (len([]rune(v.Content)) * 500) / 1000
		}
	}
	return RobotPlaybackWithTimeout(job_uuid, session_id, play, timeout)
}

func RobotPlaybackWithTimeout(job_uuid string, session_id string, play Playback, timeout int) event.Result {
	var play_str strings.Builder
	for i, v := range play.Batch {
		support := true
		switch v.Type {
		case "tts":
			play_str.WriteString("tts://")
		case "file":
			play_str.WriteString("file://")
		case "url":
		default:
			support = false
		}
		if !support {
			continue
		}
		play_str.WriteString(v.Content)
		if i != len(play.Batch)-1 {
			play_str.WriteByte('|')
		}
	}

	if len(play_str.String()) == 0 {
		return event.Result{
			Code: ERROR_CODE_PARAM_ERROR,
		}
	}
	base64_str := "base64_" + base64.StdEncoding.EncodeToString([]byte(play_str.String()))
	cmd_name := "robot_playback_ext"
	if play.LoopCount == 0 {
		play.LoopCount = 1
	}
	cmd := event.PBXCommand{
		JobId: job_uuid,
		Cmd:   cmd_name,
		Arg: format_arg(job_uuid,
			strconv.FormatBool(play.IsBreak),
			strconv.FormatBool(play.IsHangup),
			strconv.FormatBool(play.IsAsync),
			strconv.Itoa(play.LoopCount),
			base64_str),
		Uuid:    session_id,
		Timeout: timeout,
	}

	ret := event.GetDefaultBus().Request(event.TOPIC_SEND_APP_WITH_PROMISE, cmd)
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

func SessionPlayback(session_id string, file string, to_second int) event.Result {
	cmd := event.PBXCommand{
		JobId:   session_id + file + ":stop",
		Cmd:     "playback",
		Arg:     file,
		Uuid:    session_id,
		Timeout: to_second,
	}
	ret := event.GetDefaultBus().Request(event.TOPIC_SEND_APP_WITH_PROMISE, cmd)
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

func SessionPlaybackAsync(session_id string, file string) event.Result {
	cmd := event.PBXCommand{
		JobId:   session_id + file + ":start",
		Cmd:     "playback",
		Arg:     file,
		Uuid:    session_id,
		Timeout: 2000,
	}
	ret := event.GetDefaultBus().Request(event.TOPIC_SEND_APP_WITH_PROMISE, cmd)
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
