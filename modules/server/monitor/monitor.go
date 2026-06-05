package monitor

import (
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule = "Monitor"

var defaultLogger log.Logger

type Monitor struct {
	exit_sig chan bool
}

func (n *Monitor) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *Monitor) Init() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	return nil
}

func (n *Monitor) Run() error {
	data_clear_timer := time.NewTimer(time.Duration(10) * time.Second)
	go func() {
		exit := false
		for !exit {
			select {
			case <-data_clear_timer.C:
				n.data_clear()
				data_clear_timer.Reset(time.Duration(10) * time.Second)
			case <-n.exit_sig:
				exit = true
			}
		}
		data_clear_timer.Stop()
	}()
	return nil
}

func (n *Monitor) Uninit() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	return nil
}

var defaultServer *Monitor

func init() {
	defaultServer = &Monitor{}
	app.GetDefaultApp().Add(0, defaultServer)
}
