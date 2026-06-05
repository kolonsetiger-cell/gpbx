package pbx

import (
	"gitee.com/kolonse_zhjsh/gpbx/event"
)

func HangupCall(session_id string) {
	cmd := event.PBXCommand{
		JobId: session_id,
		Cmd:   "uuid_kill",
		Arg:   session_id,
		Uuid:  session_id,
	}

	event.GetDefaultBus().Publish(event.TOPIC_SEND_API, cmd)
}
