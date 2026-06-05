package app

import (
	"gitee.com/kolonse_zhjsh/gpbx/log"
)

type Module interface {
	SetLogger(log.Logger)
	Init() error
	Run() error
	Uninit() error
}
