package event

const (
	TOPIC_SESSION_CREATE        = 1
	TOPIC_SESSION_STATE         = 2
	TOPIC_SESSION_ANSWER        = 3
	TOPIC_SESSION_HANGUP        = 4
	TOPIC_SESSION_DESTROY       = 5
	TOPIC_SEND_APP              = 6
	TOPIC_SEND_API              = 7
	TOPIC_CUSTOM_PROMISE        = 8
	TOPIC_SEND_API_WITH_PROMISE = 9
	TOPIC_SEND_APP_WITH_PROMISE = 10
	TOPIC_ASR_RESULT            = 11
	TOPIC_REGISTER              = 12
	TOPIC_CALLSTATUS            = 13
	TOPIC_SESSION_DTMF          = 14
	TOPIC_SESSION_PLAYBACK_STOP = 15
)

type SessionCreate struct {
	TaskId       string `json:"task_id"`
	SessionId    string `json:"session_id"`
	Time         string `json:"time"`
	Robot        string `json:"robot"`
	RobotPartner string `json:"robot_partner"`
	Caller       string `json:"caller"`
	Callee       string `json:"callee"`
	Direction    string `json:"direction"`
}

type SessionState struct {
	TaskId    string `json:"task_id"`
	SessionId string `json:"session_id"`
	Time      string `json:"time"`
	CallState string `json:"call_state"`
}

type SessionAnswer struct {
	TaskId    string `json:"task_id"`
	SessionId string `json:"session_id"`
	Time      string `json:"time"`
	Caller    string `json:"caller"`
	Callee    string `json:"callee"`
	Direction string `json:"direction"`
}

type SessionHangup struct {
	TaskId      string `json:"task_id"`
	SessionId   string `json:"session_id"`
	Time        string `json:"time"`
	HangupCode  string `json:"hangup_code"`
	HangupCause string `json:"hangup_cause"`
	Caller      string `json:"caller"`
	Callee      string `json:"callee"`
}

type SessionDestroy struct {
	TaskId    string `json:"task_id"`
	SessionId string `json:"session_id"`
	Time      string `json:"time"`
}

type SessionDTMF struct {
	SessionId string `json:"session_id"`
	Dtmf      string `json:"dtmf"`
	Time      string `json:"time"`
}

type SessionPlaybackStart struct {
	SessionId string `json:"session_id"`
	File      string `json:"file"`
	Time      string `json:"time"`
}

type SessionPlaybackStop struct {
	SessionId string `json:"session_id"`
	File      string `json:"file"`
	Status    string `json:"status"`
	Time      string `json:"time"`
}

type PBXCommand struct {
	JobId   string `json:"job_uuid"`
	Cmd     string `json:"cmd"`
	Arg     string `json:"arg"`
	Uuid    string `json:"uuid"`
	Timeout int    `json:"timeout"`
}

type CustomPromise struct {
	JobId   string `json:"job_uuid"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

type CustomRobotAsr struct {
	TaskId    string `json:"task_id"`
	SessionId string `json:"session_id"`
	AsrState  string `json:"asr_state"`
	AsrResult string `json:"asr_result"`
	Time      string `json:"time"`
}

type CustomRegister struct {
	Username    string `json:"username"`
	Expires     string `json:"expires"`
	CallId      string `json:"call-id"`
	NetworkIP   string `json:"network-ip"`
	NetworkPort string `json:"network-port"`
}

type CustomCallStatus struct {
	Number string `json:"number"`
	Status string `json:"status"`
	Time   string `json:"time"`
}
