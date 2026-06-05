package app

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"

	"gitee.com/kolonse_zhjsh/gpbx/kcfg"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"gitee.com/kolonse_zhjsh/gpbx/store"
)

const ThisModule = "App"

type defaultLogger struct {
}

func (l *defaultLogger) Debug(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

func (l *defaultLogger) Info(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

func (l *defaultLogger) Warn(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

func (l *defaultLogger) Error(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

type App struct {
	modules      map[int][]Module
	cfg_path     string
	cfg_mu       sync.RWMutex
	cfg          *kcfg.Cfg
	exit_sig     chan os.Signal
	logger       log.Logger
	store_engine store.Store
}

func (d *App) ReloadCfg() {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			d.logger.Error(ThisModule, "Reload Cfg Failed, err:%v", err)
		}
	}()
	cfg := kcfg.NewCfg()
	cfg.ParseFile(d.cfg_path)
	d.cfg_mu.Lock()
	defer d.cfg_mu.Unlock()
	d.cfg = cfg
}

func (d *App) GetCfg() *kcfg.Cfg {
	d.cfg_mu.Lock()
	defer d.cfg_mu.Unlock()
	return d.cfg
}

func (d *App) GetStoreEngine() store.Store {
	return d.store_engine
}

func (d *App) SetLogger(logger log.Logger) {
	d.logger = logger
}

func (d *App) Init(config_path string) error {
	d.logger.Info(ThisModule, "Init App Begin")
	d.cfg_path = config_path
	d.cfg = kcfg.NewCfg()
	d.cfg.ParseFile(d.cfg_path)
	engine_str := d.cfg.Child("store.path").GetString()
	d.store_engine = store.Get(engine_str)
	if d.store_engine == nil {
		d.logger.Error(ThisModule, "Store Engine %v Not Support", engine_str)
		return errors.New("Engine Not Support")
	}
	d.store_engine.SetLogger(d.logger)
	err := d.store_engine.Load(engine_str)
	if err != nil {
		d.logger.Error(ThisModule, "Store Engine %v Load Failed, Err: %v", engine_str, err.Error())
		return err
	}
	for id, modules := range d.modules {
		d.logger.Debug(ThisModule, "Init Module ID %d", id)
		for _, m := range modules {
			m.SetLogger(d.logger)
			e := m.Init()
			if e != nil {
				return e
			}
		}
	}
	d.logger.Info(ThisModule, "Init App End")
	return nil
}

func (d *App) Uninit() error {
	d.logger.Info(ThisModule, "Uninit App Begin")
	for _, modules := range d.modules {
		for _, m := range modules {
			m.Uninit()
		}
	}
	d.store_engine.Unload()
	d.logger.Info(ThisModule, "Uninit App End")
	close(d.exit_sig)
	return nil
}

func (d *App) TriggerExit() {
	d.exit_sig <- syscall.SIGINT
}

func (d *App) Run() error {
	d.logger.Info(ThisModule, "Run App Begin")
	for _, modules := range d.modules {
		for _, m := range modules {
			e := m.Run()
			if e != nil {
				return e
			}
		}
	}
	<-d.exit_sig
	d.logger.Info(ThisModule, "Run App End")
	return nil
}

func (d *App) Add(id int, m Module) {
	_, ok := d.modules[id]
	if !ok {
		d.modules[id] = []Module{}
	}
	d.modules[id] = append(d.modules[id], m)
}

var defaultApp *App

func GetDefaultApp() *App {
	return defaultApp
}

func init() {
	defaultApp = &App{
		modules:  make(map[int][]Module),
		exit_sig: make(chan os.Signal, 1),
		logger:   &defaultLogger{},
	}
}
