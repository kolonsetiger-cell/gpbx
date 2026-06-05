package ivr

import (
	"context"

	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"github.com/google/uuid"
	"github.com/safejob/dify-sdk-go"
	"github.com/safejob/dify-sdk-go/base"
	"github.com/safejob/dify-sdk-go/types"
	// "github.com/safejob/dify-sdk-go/types"
)

type DifyConf struct {
	Url   string
	Token string `json:"token"`
}
type dify_engine struct {
	client         *base.Client
	conf           DifyConf
	conversationId string
	session_id     string
	task_id        string
	is_ok          bool
	exit           chan bool
	cancel         context.CancelFunc
}

func (l *dify_engine) pushDtmf(dtmf string) {

}

func (o *dify_engine) Close() {
	o.is_ok = false
	if o.cancel != nil {
		o.cancel()
	}
	<-o.exit
}

func (o *dify_engine) do() {
	go func() {
		var err error
		user := uuid.New().String()
		o.client, err = dify.NewClient(dify.ClientConfig{
			ApiServer: o.conf.Url,
			ApiKey:    o.conf.Token,
			User:      user,
		})
		if err != nil {
			defaultLogger.Error(ThisModule, "Dify NewClient err: %v", err)
		} else {
			ctx, cancel := context.WithCancel(context.Background())
			o.cancel = cancel
			ev := o.client.WorkflowApp().Run(ctx, types.WorkflowRequest{
				Inputs: map[string]any{
					"session_id": o.session_id,
					"task_id":    o.task_id,
				},
				ResponseMode: "streaming",
			}).ParseToEventCh()
			is_end := false
			task_id := ""
			for !is_end {
				msg := <-ev
				defaultLogger.Info(ThisModule, "Dify event: %v", msg)
				switch msg.Type {
				case types.EVENT_WORKFLOW_STARTED:
					data, ok := msg.Data.(*types.EventWorkflowStarted)
					if ok {
						task_id = data.TaskId
					} else {
						defaultLogger.Error(ThisModule, "Dify event: %v", msg)
						is_end = true
					}
				case types.EVENT_WORKFLOW_FINISHED:
					is_end = true
				case types.EVENT_ERROR:
					err, ok := msg.Data.(*types.EventError)
					if ok {
						defaultLogger.Error(ThisModule, "Dify event: %v", err)
					}
					is_end = true
				}
				if !o.is_ok {
					is_end = true
				}
			}
			if task_id != "" {
				_ = o.client.WorkflowApp().Stop(task_id, user)
			}
		}
		if o.is_ok {
			pbx.HangupCall(o.session_id)
		}

		o.is_ok = false
		o.exit <- true
		close(o.exit)
	}()
}

func NewDifyEngine(difyConf DifyConf, session_id, task_id string) *dify_engine {
	return &dify_engine{
		conf:       difyConf,
		session_id: session_id,
		task_id:    task_id,
		is_ok:      true,
		exit:       make(chan bool, 10),
	}
}
