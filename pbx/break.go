package pbx

import (
	"gitee.com/kolonse_zhjsh/gpbx/event"
)

func RobotBreak(session_id string) {
	cmd := event.PBXCommand{
		JobId:   "",
		Cmd:     "play_break",
		Arg:     "",
		Uuid:    session_id,
		Timeout: -1,
	}

	event.GetDefaultBus().Publish(event.TOPIC_SEND_APP, cmd)
}

func SessionBreak(session_id string) {
	cmd := event.PBXCommand{
		JobId:   "",
		Cmd:     "break",
		Arg:     "",
		Uuid:    session_id,
		Timeout: -1,
	}

	event.GetDefaultBus().Publish(event.TOPIC_SEND_APP, cmd)
}

func ApiBreak(session_id string) {
	cmd := event.PBXCommand{
		JobId:   "",
		Cmd:     "uuid_break",
		Arg:     session_id,
		Uuid:    session_id,
		Timeout: -1,
	}

	event.GetDefaultBus().Publish(event.TOPIC_SEND_API, cmd)
}
