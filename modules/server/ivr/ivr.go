package ivr

import (
	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule = "IVR"

var defaultLogger log.Logger

// type ai_vendor interface {
// 	Load(model, url, token string, max_history int) error
// 	Say(prompt string, text string, timeout_ms int) (string, error)
// }

type IVRModule struct {
	exit_sig chan bool
}

func (n *IVRModule) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *IVRModule) Init() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	return nil
}

func (n *IVRModule) Run() error {
	sess_destroy_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_DESTROY)
	sess_answer_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_ANSWER)
	dtmf_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_DTMF)
	go func() {
		exit := false
		for !exit {
			select {
			case msg := <-sess_destroy_chan:
				destroy_msg := msg.Data.(*event.SessionDestroy)
				defaultLogger.Info(ThisModule, "Session:%v Destroy", destroy_msg)
				body := defaultManager.Get(destroy_msg.SessionId)
				if body != nil {
					body.on_hangup()
				}
				defaultManager.Delete(destroy_msg.SessionId)
			case msg := <-sess_answer_chan:
				answer_msg := msg.Data.(*event.SessionAnswer)
				session := datacenter.GetSessionManager().Get(answer_msg.SessionId)
				if session == nil {
					continue
				}
				task := datacenter.GetTaskManager().Get(session.TaskID)
				if task == nil {
					continue
				}
				if task.TaskType != datacenter.TASK_TYPE_originate_to_ivr {
					continue
				}
				defaultLogger.Info(ThisModule, "Session:%v Answer", answer_msg)
				body := newIVRBody()
				body.task_id = answer_msg.SessionId
				body.session_id = answer_msg.SessionId
				body.ivr_id = task.ToIvrID
				defaultManager.Set(answer_msg.TaskId, body)
				body.on_answer()
			case msg := <-dtmf_chan:
				dtmf, ok := msg.Data.(*event.SessionDTMF)
				if ok {
					session := datacenter.GetSessionManager().Get(dtmf.SessionId)
					if session == nil {
						continue
					}
					task := datacenter.GetTaskManager().Get(session.TaskID)
					if task != nil {
						defaultLogger.Info(ThisModule, "Task:%v", task)
						body := defaultManager.Get(task.TaskID)
						if body != nil {
							defaultLogger.Info(ThisModule, "IVR Body:%v", body)
							body.on_dtmf(dtmf.Dtmf)
						} else {
							defaultLogger.Error(ThisModule, "IVR Body:%v", body)
						}
					}
				}
			case <-n.exit_sig:
				exit = true
			}
		}
	}()
	return nil
}

func (n *IVRModule) Uninit() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	return nil
}

var defaultServer *IVRModule

func init() {
	defaultServer = &IVRModule{}
	app.GetDefaultApp().Add(0, defaultServer)
}
