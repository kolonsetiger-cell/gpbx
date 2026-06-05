package auth

import (
	"os"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule = "Auth"
const EXPIRE_TIME = "2026-06-10 00:00:00"

var defaultLogger log.Logger

type Auth struct {
	exit_sig chan bool
}

func (n *Auth) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *Auth) Init() error {
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	return nil
}

func (n *Auth) Run() error {
	defaultLogger.Info(ThisModule, "Run Auth")
	go func() {
		exit := false
		timer := time.NewTicker(time.Second * 60)
		defer timer.Stop()
		for !exit {
			select {
			case <-timer.C:
				// 验证时间: 与 content[1] 相差不超过10分钟
				signTime, err := time.ParseInLocation("2006-01-02 15:04:05", EXPIRE_TIME, time.Local)
				if err != nil {
					panic(err)
				}
				now := time.Now()
				diff := now.Sub(signTime)
				if diff > 0 {
					defaultLogger.Error(ThisModule, "授权过期")
					os.Exit(-1)
				}
			case <-n.exit_sig:
				exit = true
			}
		}
	}()
	return nil
}

func (n *Auth) Uninit() error {
	defaultLogger.Info(ThisModule, "Uninit Auth")
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	return nil
}

var defaultServer *Auth

func init() {
	defaultServer = &Auth{}
	app.GetDefaultApp().Add(0, defaultServer)
}
