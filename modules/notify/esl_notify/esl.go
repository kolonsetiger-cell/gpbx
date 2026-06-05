package esl_notify

import (
	"strconv"
	"sync"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
	"gitee.com/kolonse_zhjsh/gpbx/esl"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule string = "EslNotify"

type promise_obj struct {
	sig      chan any
	timer    *time.Timer
	mu       sync.Mutex
	released bool
}

func (obj *promise_obj) done(data any) {
	obj.mu.Lock()
	defer obj.mu.Unlock()
	if obj.released {
		return
	}
	obj.released = true
	obj.sig <- data
	close(obj.sig)
	if obj.timer != nil {
		obj.timer.Stop()
	}
}

func NewPromiseObj(to int, to_callback func()) *promise_obj {
	obj := &promise_obj{
		sig:      make(chan any),
		released: false,
	}

	if to > 0 {
		obj.timer = time.NewTimer(time.Duration(to) * time.Second)
		go func() {
			<-obj.timer.C
			obj.mu.Lock()
			if !obj.released {
				close(obj.sig)
				obj.timer.Stop()
				obj.released = true
			}
			obj.mu.Unlock()
			to_callback()
		}()
	}
	return obj
}

type EslNotify struct {
	logger   log.Logger
	exit_sig chan bool
	esl      *esl.Client
	mu       sync.RWMutex
	promises map[string]*promise_obj
}

func (c *EslNotify) SetLogger(logger log.Logger) {
	c.logger = logger
}

func (c *EslNotify) send_app(msg *event.Message) {
	cmd := msg.Data.(event.PBXCommand)
	err := c.esl.SendAPP(cmd.Cmd, cmd.Arg, cmd.Uuid)
	if err != nil {
		msg.Done(-1, err.Error())
		return
	}
	msg.Done(0, nil)
}

func (c *EslNotify) send_api(msg *event.Message) {
	cmd := msg.Data.(event.PBXCommand)
	err := c.esl.SendAPI(cmd.Cmd, cmd.Arg, cmd.Uuid)
	if err != nil {
		msg.Done(-1, err.Error())
		return
	}
	msg.Done(0, nil)
}

func (c *EslNotify) send_api_with_promise(msg *event.Message) {
	cmd := msg.Data.(event.PBXCommand)
	err := c.esl.SendAPI(cmd.Cmd, cmd.Arg, cmd.Uuid)
	if err != nil {
		msg.Done(-1, err.Error())
		return
	}

	c.mu.Lock()
	if obj, ok := c.promises[cmd.JobId]; ok {
		obj.done(nil)
		delete(c.promises, cmd.JobId)
	}
	obj := NewPromiseObj(cmd.Timeout, func() {
		msg.Done(-1, "Timeout")
		c.mu.Lock()
		delete(c.promises, cmd.JobId)
		c.mu.Unlock()
	})
	c.promises[cmd.JobId] = obj
	go func() {
		pro_data := <-obj.sig
		msg.Done(0, pro_data)
	}()
	c.mu.Unlock()
}

func (c *EslNotify) send_app_with_promise(msg *event.Message) {
	cmd := msg.Data.(event.PBXCommand)
	err := c.esl.SendAPP(cmd.Cmd, cmd.Arg, cmd.Uuid)
	if err != nil {
		msg.Done(-1, err.Error())
		return
	}

	c.mu.Lock()
	if obj, ok := c.promises[cmd.JobId]; ok {
		obj.done(nil)
		delete(c.promises, cmd.JobId)
	}
	obj := NewPromiseObj(cmd.Timeout, func() {
		msg.Done(-1, "Timeout")
		c.mu.Lock()
		delete(c.promises, cmd.JobId)
		c.mu.Unlock()
	})
	c.promises[cmd.JobId] = obj
	c.logger.Debug(ThisModule, "%v Send App With Promise %v", cmd.JobId, cmd)
	go func() {
		pro_data := <-obj.sig
		c.logger.Debug(ThisModule, "%v Recv Response %v", cmd.JobId, pro_data)
		msg.Done(0, pro_data)
	}()
	c.mu.Unlock()
}

func (c *EslNotify) Init() error {
	c.logger.Debug(ThisModule, "Init Begin")
	c.esl = esl.NewClient()
	c.esl.SetLogger(c.logger)
	c.esl.On("CHANNEL_CREATE", func(k string, pck *esl.Package) {
		data := &event.SessionCreate{
			TaskId:       pck.GetBody("variable_task_id"),
			SessionId:    pck.GetBody("Unique-ID"),
			Time:         pck.GetBody("Event-Date-Timestamp"),
			Robot:        pck.GetBody("variable_robot"),
			RobotPartner: pck.GetBody("variable_robot_partner"),
			Caller:       pck.GetBody("Caller-Caller-ID-Number"),
			Callee:       pck.GetBody("Caller-Destination-Number"),
			Direction:    pck.GetBody("Call-Direction"),
		}
		t, _ := strconv.Atoi(data.Time)
		gsession := datacenter.GetGlobalSessionManager().CreateAndGet(data.SessionId)
		gsession.CallTime = int64(t)
		gsession.SessionID = data.SessionId
		gsession.IsRobot = data.Robot == "true"
		gsession.Caller = data.Caller
		gsession.Callee = data.Callee
		gsession.Direction = data.Direction
		c.logger.Debug(ThisModule, "Session %v Create %v, Is Robot:%v", data.SessionId, gsession.CallTime, gsession.IsRobot)
		session := datacenter.GetSessionManager().Get(data.SessionId)
		if session != nil {
			session.CallTime = int64(t)
			session.IsRobot = data.Robot == "true"
		}
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_CREATE, data)
	})
	c.esl.On("CHANNEL_CALLSTATE", func(k string, pck *esl.Package) {
		data := &event.SessionState{
			TaskId:    pck.GetBody("variable_task_id"),
			SessionId: pck.GetBody("Unique-ID"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
			CallState: pck.GetBody("Channel-Call-State"),
		}

		if data.CallState == "RINGING" {
			t, _ := strconv.Atoi(data.Time)
			gsession := datacenter.GetGlobalSessionManager().CreateAndGet(data.SessionId)
			if gsession != nil {
				gsession.RingTime = int64(t)
			}
			c.logger.Debug(ThisModule, "Session %v Ring %v", data.SessionId, t)
			session := datacenter.GetSessionManager().Get(data.SessionId)
			if session != nil {
				session.RingTime = int64(t)
			}
		}
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_STATE, data)
	})
	c.esl.On("CHANNEL_ANSWER", func(k string, pck *esl.Package) {
		data := &event.SessionAnswer{
			TaskId:    pck.GetBody("variable_task_id"),
			SessionId: pck.GetBody("Unique-ID"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
			Caller:    pck.GetBody("Caller-Caller-ID-Number"),
			Callee:    pck.GetBody("Caller-Destination-Number"),
			Direction: pck.GetBody("Call-Direction"),
		}
		t, _ := strconv.Atoi(data.Time)
		gsession := datacenter.GetGlobalSessionManager().CreateAndGet(data.SessionId)
		if gsession != nil {
			gsession.AnswerTime = int64(t)
		}
		c.logger.Debug(ThisModule, "Session %v Answer %v", data.SessionId, gsession.AnswerTime)
		session := datacenter.GetSessionManager().Get(data.SessionId)
		if session != nil {
			session.AnswerTime = int64(t)
		}
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_ANSWER, data)
	})
	c.esl.On("CHANNEL_HANGUP_COMPLETE", func(k string, pck *esl.Package) {
		data := &event.SessionHangup{
			TaskId:      pck.GetBody("variable_task_id"),
			SessionId:   pck.GetBody("Unique-ID"),
			Time:        pck.GetBody("Event-Date-Timestamp"),
			HangupCode:  pck.GetBody("variable_sip_term_status"),
			HangupCause: pck.GetBody("variable_hangup_cause"),
			Caller:      pck.GetBody("Caller-Caller-ID-Number"),
			Callee:      pck.GetBody("Caller-Destination-Number"),
		}
		t, _ := strconv.Atoi(data.Time)
		gsession := datacenter.GetGlobalSessionManager().CreateAndGet(data.SessionId)
		if gsession != nil {
			gsession.HangupTime = int64(t)
		}
		c.logger.Debug(ThisModule, "Session %v Hangup %v", data.SessionId, gsession.HangupTime)
		session := datacenter.GetSessionManager().Get(data.SessionId)
		if session != nil {
			session.HangupTime = int64(t)
		}
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_HANGUP, data)
	})
	c.esl.On("CHANNEL_DESTROY", func(k string, pck *esl.Package) {
		data := &event.SessionDestroy{
			TaskId:    pck.GetBody("variable_task_id"),
			SessionId: pck.GetBody("Unique-ID"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
		}
		t, _ := strconv.Atoi(data.Time)
		gsession := datacenter.GetGlobalSessionManager().CreateAndGet(data.SessionId)
		if gsession != nil {
			gsession.DestroyTime = int64(t)
		}
		c.logger.Debug(ThisModule, "Session %v Destroy %v", data.SessionId, gsession.DestroyTime)
		session := datacenter.GetSessionManager().Get(data.SessionId)
		if session != nil {
			session.DestroyTime = int64(t)
		}
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_DESTROY, data)
	})
	c.esl.On("DTMF", func(k string, pck *esl.Package) {
		data := &event.SessionDTMF{
			SessionId: pck.GetBody("Unique-ID"),
			Dtmf:      pck.GetBody("DTMF-Digit"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
		}
		c.logger.Debug(ThisModule, "Session %v Recv Dtmf %v", data.SessionId, data)
		session := datacenter.GetSessionManager().Get(data.SessionId)
		if session != nil {
			session.PushDtmf(data.Dtmf)
		}
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_DTMF, data)
	})
	c.esl.On("PLAYBACK_START", func(k string, pck *esl.Package) {
		data := &event.SessionPlaybackStart{
			SessionId: pck.GetBody("Unique-ID"),
			File:      pck.GetBody("Playback-File-Path"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
		}
		c.logger.Debug(ThisModule, "Session %v Recv Playback Start %v", data.SessionId, data)
		c.mu.Lock()
		if obj, ok := c.promises[data.SessionId+data.File+":start"]; ok {
			obj.done(&event.CustomPromise{
				JobId: data.SessionId + data.File,
				Code:  0,
				Time:  data.Time,
			})
			delete(c.promises, data.SessionId+data.File)
		}
		c.mu.Unlock()
	})
	c.esl.On("PLAYBACK_STOP", func(k string, pck *esl.Package) {
		data := &event.SessionPlaybackStop{
			SessionId: pck.GetBody("Unique-ID"),
			File:      pck.GetBody("Playback-File-Path"),
			Status:    pck.GetBody("Playback-Status"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
		}
		c.logger.Debug(ThisModule, "Session %v Recv Playback Stop %v", data.SessionId, data)
		event.GetDefaultBus().Publish(event.TOPIC_SESSION_PLAYBACK_STOP, data)
		c.mu.Lock()
		if obj, ok := c.promises[data.SessionId+data.File+":stop"]; ok {
			obj.done(&event.CustomPromise{
				JobId:   data.SessionId + data.File,
				Code:    0,
				Message: data.Status,
				Time:    data.Time,
			})
			delete(c.promises, data.SessionId+data.File)
		}
		c.mu.Unlock()
	})
	c.esl.OnCustom("cus_event::promise", func(k string, pck *esl.Package) {
		c.logger.Debug(ThisModule, "Promise %v", pck.Dump())
		data := &event.CustomPromise{
			JobId:   pck.GetBody("cus_event_job_id"),
			Code:    pck.GetBodyAsInt("cus_event_code"),
			Message: pck.GetBody("cus_event_message"),
			Time:    pck.GetBody("Event-Date-Timestamp"),
		}

		c.mu.Lock()
		if obj, ok := c.promises[data.JobId]; ok {
			obj.done(data)
			delete(c.promises, data.JobId)
		}
		c.mu.Unlock()
	})
	c.esl.OnCustom("robot::asr", func(k string, pck *esl.Package) {
		c.logger.Debug(ThisModule, "Asr %v", pck.Dump())
		data := &event.CustomRobotAsr{
			SessionId: pck.GetBody("unique-id"),
			AsrState:  pck.GetBody("asr-state"),
			AsrResult: pck.GetBody("asr-result"),
			Time:      pck.GetBody("Event-Date-Timestamp"),
		}
		session := datacenter.GetSessionManager().Get(data.SessionId)
		if session != nil {
			task := datacenter.GetTaskManager().Get(session.TaskID)
			if task != nil {
				if !task.IsLocked() {
					data.TaskId = task.TaskID
					c.logger.Info(ThisModule, "<%v> Asr Session %v Recv %v", task.TaskID, data.SessionId, data.AsrResult)
					event.GetDefaultBus().Publish(event.TOPIC_ASR_RESULT, data)
				} else {
					c.logger.Debug(ThisModule, "<%v> Asr Session %v Locked, Ignore", task.TaskID, data.SessionId)
				}
			} else {
				c.logger.Debug(ThisModule, "Asr Session %v Not Found Task, Ignore", data.SessionId)
			}
		} else {
			c.logger.Debug(ThisModule, "Asr Session %v Not Found Session, Ignore", data.SessionId)
		}
	})
	c.esl.OnCustom("sofia::register", func(k string, pck *esl.Package) {
		// c.logger.Debug(ThisModule, "Register %v", pck.Dump())
		data := &event.CustomRegister{
			Username:    pck.GetBody("username"),
			Expires:     pck.GetBody("expires"),
			CallId:      pck.GetBody("call-id"),
			NetworkIP:   pck.GetBody("network-ip"),
			NetworkPort: pck.GetBody("network-port"),
		}
		event.GetDefaultBus().Publish(event.TOPIC_REGISTER, data)
	})
	c.esl.OnCustom("sofia::unregister", func(k string, pck *esl.Package) {
		// c.logger.Debug(ThisModule, "Unegister %v", pck.Dump())
		data := &event.CustomRegister{
			Username: pck.GetBody("username"),
			Expires:  "0",
			CallId:   pck.GetBody("call-id"),
		}
		event.GetDefaultBus().Publish(event.TOPIC_REGISTER, data)
	})
	c.esl.OnCustom("cus_event::callstatus", func(k string, pck *esl.Package) {
		c.logger.Debug(ThisModule, "Callstatus %v", pck.Dump())
		data := &event.CustomCallStatus{
			Number: pck.GetBody("number"),
			Status: pck.GetBody("status"),
			Time:   pck.GetBody("Event-Date-Timestamp"),
		}
		event.GetDefaultBus().Publish(event.TOPIC_CALLSTATUS, data)
	})
	c.esl.On("HEARTBEAT", nil)
	c.esl.OnEventCallback(func(s *esl.Client, state int) {
		// log.NormalLogger.Debug(ThisModule, "%v", state)
		switch state {
		case esl.ESLClientState_Success:
			// esl success
		case esl.ESLClientState_Null:
			// esl lost
		default:

		}
	})
	if c.exit_sig != nil {
		close(c.exit_sig)
	}
	c.exit_sig = make(chan bool, 1)
	c.logger.Debug(ThisModule, "Init End")
	return nil
}

func (c *EslNotify) Uninit() error {
	c.logger.Debug(ThisModule, "Uninit Begin")
	c.esl.Disconnect()
	if c.exit_sig != nil {
		close(c.exit_sig)
		c.exit_sig = nil
	}
	c.logger.Debug(ThisModule, "Uninit End")
	return nil
}

func (c *EslNotify) Run() error {
	cfg := app.GetDefaultApp().GetCfg()
	config := esl.FSConfig{
		Addr:                      cfg.Child("freeswitch.addr").GetString(),
		User:                      cfg.Child("freeswitch.user").GetString(),
		Pass:                      cfg.Child("freeswitch.pass").GetString(),
		Reconnect_Interval_Second: int(cfg.Child("freeswitch.Reconnect_Interval_Second").GetInt()),
	}

	c.esl.Connect(config)
	// 订阅 ESL 消息
	app_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SEND_APP)
	api_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SEND_API)
	api_chan_with_promise := event.GetDefaultBus().Subscribe(event.TOPIC_SEND_API_WITH_PROMISE)
	app_chan_with_promise := event.GetDefaultBus().Subscribe(event.TOPIC_SEND_APP_WITH_PROMISE)

	go func() {
		c.logger.Debug(ThisModule, "Send ESL Start")
		exit := false
		for !exit {
			select {
			case msg := <-app_chan:
				c.send_app(msg)
			case msg := <-api_chan:
				c.send_api(msg)
			case msg := <-api_chan_with_promise:
				c.send_api_with_promise(msg)
			case msg := <-app_chan_with_promise:
				c.send_app_with_promise(msg)
			case <-c.exit_sig:
				exit = true
			}
		}
		c.logger.Debug(ThisModule, "Send ESL End")
	}()
	return nil
}

func init() {
	app.GetDefaultApp().Add(0, &EslNotify{
		promises: make(map[string]*promise_obj),
	})
}
