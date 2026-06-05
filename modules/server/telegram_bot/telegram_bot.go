package telegram_bot

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const ThisModule = "TelegramBot"

var defaultLogger log.Logger

type TelegramBotConfig struct {
	ServerPort int64  `json:"server_port"`
	Token      string `json:"token"`
	Enable     bool   `json:"enable"`
	BindScript string `json:"bind_script"`
	Proxy      string `json:"proxy"`
	ToIvrId    string `json:"to_ivr_id"`
	Number     string `json:"number"`
	TenantId   string `json:"tenant_id"`
	Timeout    int64  `json:"timeout"`
}

type TelegramBotModule struct {
	exit_sig chan bool
	web      *gin.Engine
	l        net.Listener
	root_bot *BotContext
}

func (m *TelegramBotModule) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (m *TelegramBotModule) api_check(c *gin.Context) {
	var req map[string]any
	_ = c.ShouldBindBodyWithJSON(&req)
	session_id, ok := req["session_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	body := globalBotManager.Get(session_id)
	if body == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id not found"})
		return
	}
	code := body.OnCheck()
	c.JSON(http.StatusOK, gin.H{"code": code})
}

func (m *TelegramBotModule) api_ensure(c *gin.Context) {
	var req map[string]any
	_ = c.ShouldBindBodyWithJSON(&req)
	session_id, ok := req["session_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	body := globalBotManager.Get(session_id)
	if body == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id not found"})
		return
	}
	body.OnEnsure()
	c.JSON(http.StatusOK, gin.H{"code": 0})
}
func (m *TelegramBotModule) Init() error {
	if m.exit_sig != nil {
		close(m.exit_sig)
	}
	m.exit_sig = make(chan bool, 1)

	cfg := app.GetDefaultApp().GetCfg()
	teleCfg := TelegramBotConfig{}
	teleCfg.Token = cfg.Child("telegram_bot.token").GetString()
	teleCfg.Enable = cfg.Child("telegram_bot.enable").GetBool()
	teleCfg.BindScript = cfg.Child("telegram_bot.bind_script").GetString()
	teleCfg.Proxy = cfg.Child("telegram_bot.proxy").GetString()
	teleCfg.ToIvrId = cfg.Child("telegram_bot.to_ivrid").GetString()
	teleCfg.Number = cfg.Child("telegram_bot.number").GetString()
	teleCfg.TenantId = cfg.Child("telegram_bot.tenant_id").GetString()
	teleCfg.Timeout = cfg.Child("telegram_bot.tenant_id").GetInt()
	teleCfg.ServerPort = cfg.Child("telegram_bot.server_port").GetInt()
	if teleCfg.Token == "" || !teleCfg.Enable {
		defaultLogger.Warn(ThisModule, "telegram_bot.token not configured, module will not start")
		return nil
	}
	m.root_bot = NewBotContext()
	err := m.root_bot.Start(teleCfg)
	if err != nil {
		m.root_bot = nil
		defaultLogger.Warn(ThisModule, "Telegram_bot Start Failed, err:%v", err)
		return err
	}
	m.web = gin.Default()
	allowedOrigins := app.GetDefaultApp().GetCfg().Child("http.cors.allowedOrigins").GetString()
	if allowedOrigins == "" {
		allowedOrigins = "*" // 默认为宽松配置
	}
	corsConfig := cors.Config{
		AllowOrigins:     []string{allowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: allowedOrigins != "*", // 允许凭证时不允许使用通配符
		MaxAge:           12 * time.Hour,
	}
	m.web.Use(cors.New(corsConfig))
	m.web.POST("/api/check", m.api_check)
	m.web.POST("/api/ensure", m.api_ensure)
	host := app.GetDefaultApp().GetCfg().Child("telegram_bot.server_host").GetString()
	port := app.GetDefaultApp().GetCfg().Child("telegram_bot.server_port").GetInt()
	if len(host) == 0 {
		host = "0.0.0.0"
	}
	if port <= 0 || port > 65536 {
		port = 8080
	}
	addr := fmt.Sprintf("%v:%v", host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	m.l = ln
	defaultLogger.Info(ThisModule, "TelegramBot config loaded, enable=%v", teleCfg.Enable)
	return nil
}

func (m *TelegramBotModule) Run() error {
	if m.root_bot == nil {
		return nil
	}

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
				body := globalBotManager.GetAndDelete(destroy_msg.SessionId)
				if body != nil {
					body.OnHangup()
				}
			case msg := <-sess_answer_chan:
				answer_msg := msg.Data.(*event.SessionAnswer)
				body := globalBotManager.Get(answer_msg.SessionId)
				if body != nil {
					body.OnAnswer()
				}

				defaultLogger.Info(ThisModule, "Session:%v Answer", answer_msg)
			case msg := <-dtmf_chan:
				dtmf, ok := msg.Data.(*event.SessionDTMF)
				if ok {
					body := globalBotManager.Get(dtmf.SessionId)
					if body != nil {
						body.OnDtmf(dtmf.Dtmf)
					}
				}
			case <-m.exit_sig:
				exit = true
			}
		}
	}()
	go func() {
		_ = m.web.RunListener(m.l)
	}()
	return nil
}

func (m *TelegramBotModule) Uninit() error {
	if m.root_bot != nil {
		m.root_bot.Stop()
		m.root_bot = nil
	}
	if m.exit_sig != nil {
		close(m.exit_sig)
		m.exit_sig = nil
	}
	if m.web != nil {
		_ = m.l.Close()
	}
	defaultLogger.Info(ThisModule, "TelegramBot stopped")
	return nil
}

var defaultServer *TelegramBotModule

func init() {
	defaultServer = &TelegramBotModule{}
	app.GetDefaultApp().Add(0, defaultServer)
}
