package ai

import (
	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule = "AI"

/**
 * AI 智能体模块
 * 1. 监听 robot session 创建事件，调用接口进行播放指定欢迎语，并创建房间
 * 2. 收到 ASR 结果，调用大模型接口
 * 3. 大模型接口返回意图和参数
 * 4. 调用 AI 流程，并返回播放的音频
 */

var defaultLogger log.Logger

type ai_vendor interface {
	Load(model, url, token string, max_history int) error
	Say(prompt string, text string, timeout_ms int) (string, error)
}

type AI struct {
	exit_sig chan bool
}

func (n *AI) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *AI) Init() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	return nil
}

func (n *AI) Run() error {
	sess_destroy_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_DESTROY)
	sess_answer_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_ANSWER)
	asr_chan := event.GetDefaultBus().Subscribe(event.TOPIC_ASR_RESULT)
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
				if session == nil || !session.IsRobot {
					continue
				}
				task := datacenter.GetTaskManager().Get(session.TaskID)
				if task == nil {
					continue
				}
				if task.TaskType != datacenter.TASK_TYPE_originate_to_robot_ai &&
					task.TaskType != datacenter.TASK_TYPE_callin_to_robot_ai &&
					task.TaskType != datacenter.TASK_TYPE_transfer_to_robot_ai {
					continue
				}
				defaultLogger.Info(ThisModule, "Session:%v Answer", answer_msg)
				body := newAIBody()
				body.task_id = answer_msg.SessionId
				body.session_id = answer_msg.SessionId
				body.robot_id = session.RobotID
				defaultManager.Set(answer_msg.TaskId, body)
				body.on_answer()
			case msg := <-asr_chan:
				{
					asr, ok := msg.Data.(*event.CustomRobotAsr)
					if ok {
						session := datacenter.GetSessionManager().Get(asr.SessionId)
						if session == nil {
							continue
						}
						task := datacenter.GetTaskManager().Get(session.TaskID)
						if task != nil {
							defaultLogger.Info(ThisModule, "Task:%v", task)
							body := defaultManager.Get(asr.TaskId)
							if body != nil {
								defaultLogger.Info(ThisModule, "AI Body:%v", body)
								body.on_asr(asr.AsrResult)
							} else {
								defaultLogger.Error(ThisModule, "AI Body:%v", body)
							}
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

func (n *AI) Uninit() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	return nil
}

var defaultServer *AI

func init() {
	defaultServer = &AI{}
	app.GetDefaultApp().Add(0, defaultServer)
}
