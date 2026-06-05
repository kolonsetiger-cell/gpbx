package http_notify

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule = "HttpNotify"

type HttpNotify struct {
	logger   log.Logger
	exit_sig chan bool
}

func (n *HttpNotify) SetLogger(logger log.Logger) {
	n.logger = logger
}

func (n *HttpNotify) Init() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	return nil
}

func (n *HttpNotify) notify(url string, msg *event.Message) {
	defer msg.Done(0, nil)
	if len(url) == 0 {
		n.logger.Info(ThisModule, "Discard Topic:%v", msg.Topic)
		return
	}
	timeout := app.GetDefaultApp().GetCfg().Child("backend.timeout").GetInt()
	if timeout <= 0 {
		timeout = 10
	}

	client := &http.Client{
		Timeout: time.Second * time.Duration(timeout), // 10秒超时
	}
	body, _ := json.Marshal(msg.Data)
	req, _ := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Golang GPBX")
	resp, err := client.Do(req)
	if err != nil {
		n.logger.Error(ThisModule, "Send to <%v> failed, body <%s>, reason <%s>", url, body, err.Error())
		return
	}
	defer resp.Body.Close()
	bosy_resp, _ := io.ReadAll(resp.Body)
	n.logger.Info(ThisModule, "Send to <%v> Completed, body <%s>, result <%s>", url, body, string(bosy_resp))
}

func (n *HttpNotify) Run() error {
	sess_create_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_CREATE)
	sess_answer_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_ANSWER)
	sess_state_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_STATE)
	sess_hangup_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_HANGUP)
	sess_destroy_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_DESTROY)
	asr_chan := event.GetDefaultBus().Subscribe(event.TOPIC_ASR_RESULT)
	go func() {
		n.logger.Debug(ThisModule, "Http Notify Start")
		exit := false
		for !exit {
			select {
			case msg := <-sess_create_chan:
				n.notify(app.GetDefaultApp().GetCfg().Child("backend.session_create").GetString(), msg)
			case msg := <-sess_answer_chan:
				n.notify(app.GetDefaultApp().GetCfg().Child("backend.session_answer").GetString(), msg)
			case msg := <-sess_state_chan:
				n.notify(app.GetDefaultApp().GetCfg().Child("backend.session_state").GetString(), msg)
			case msg := <-sess_hangup_chan:
				n.notify(app.GetDefaultApp().GetCfg().Child("backend.session_hangup").GetString(), msg)
			case msg := <-sess_destroy_chan:
				n.notify(app.GetDefaultApp().GetCfg().Child("backend.session_destroy").GetString(), msg)
			case msg := <-asr_chan:
				{
					asr, ok := msg.Data.(*event.CustomRobotAsr)
					if ok {
						// session := datacenter.GetSessionManager().Get(asr.SessionId)
						task := datacenter.GetTaskManager().Get(asr.TaskId)
						if task != nil {
							url := task.AsrCallback
							if len(url) == 0 {
								url = app.GetDefaultApp().GetCfg().Child("asr.callback").GetString()
							}
							if task.TaskType != datacenter.TASK_TYPE_originate_to_robot_vendor &&
								task.TaskType != datacenter.TASK_TYPE_callin_to_robot_vendor &&
								task.TaskType != datacenter.TASK_TYPE_transfer_to_robot_vendor {
								continue
							}
							n.notify(url, msg)
						}
					}
				}
			case <-n.exit_sig:
				exit = true
			}
		}
		n.logger.Debug(ThisModule, "Http Notify End")
	}()
	return nil
}

func (n *HttpNotify) Uninit() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	return nil
}

func init() {
	app.GetDefaultApp().Add(10, &HttpNotify{})
}
