package callcenter_agent

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"gitee.com/kolonse_zhjsh/gpbx/modules/server/monitor"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const ThisModule = "CallCenter_Agent"

var defaultLogger log.Logger

type HttpServer struct {
	web         *gin.Engine
	l           net.Listener
	uninit_flag bool
	exit_sig    chan bool
}

func (n *HttpServer) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *HttpServer) Init() error {
	monitor.Register(func() {
		cleared := store.ExtensionManagerInstance.ClearExpire()
		for _, e := range cleared {
			pos := strings.Index(e.Number, "-")
			if pos == -1 {
				continue
			}
			tenantId := e.Number[0:pos]
			extId := e.Number[pos+1:]
			defaultLogger.Info(ThisModule, "Extension %v %v Expire, Clear", tenantId, extId)
			_ = app.GetDefaultApp().GetStoreEngine().StoreExtension(store.Extension{
				TenantId:    tenantId,
				ExtensionId: extId,
				Status:      store.EXTENSION_STATUS_OFFLINE,
			})
		}
	})
	n.web = gin.Default()
	if n.exit_sig != nil {
		close(n.exit_sig)
	}
	n.exit_sig = make(chan bool, 1)
	// 配置 CORS 中间件
	// 从配置读取允许的来源，默认仅允许同源
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
	n.web.Use(cors.New(corsConfig))

	upGrader = websocket.Upgrader{
		// 允许跨域（开发环境用，生产建议根据需求限制）
		CheckOrigin: func(r *http.Request) bool {
			if allowedOrigins == "*" {
				return true
			}
			return strings.Contains(allowedOrigins, r.Header.Get("Origin"))
		},
	}
	n.web.GET("/agent", wsHandler)
	for r, f := range routers {
		n.web.POST(r, f)
	}
	api := n.web.Group("/api")
	for _, r := range registers {
		defaultLogger.Info(ThisModule, "Register %v", r)
		r(api)
	}
	for _, s := range statics {
		s(n.web)
	}
	host := app.GetDefaultApp().GetCfg().Child("http.host").GetString()
	port := app.GetDefaultApp().GetCfg().Child("http.agent_port").GetInt()
	if len(host) == 0 {
		host = "0.0.0.0"
	}
	if port <= 0 || port > 65536 {
		port = 8081
	}
	addr := fmt.Sprintf("%v:%v", host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	n.l = ln
	n.uninit_flag = false
	defaultLogger.Info(ThisModule, "Http Server Listen addr <%v:%v>", host, port)
	return nil
}

func (n *HttpServer) Run() error {
	go func() {
		err := n.web.RunListener(n.l)
		if !n.uninit_flag && err != nil {
			defaultLogger.Error(ThisModule, "Http Server err:%v", err.Error())
			panic(err)
		}
	}()

	register_chan := event.GetDefaultBus().Subscribe(event.TOPIC_REGISTER)
	callstatus_chan := event.GetDefaultBus().Subscribe(event.TOPIC_CALLSTATUS)
	hangup_chan := event.GetDefaultBus().Subscribe(event.TOPIC_SESSION_HANGUP)
	go func() {
		exit := false
		for !exit {
			select {
			case <-n.exit_sig:
				exit = true
			case msg := <-register_chan:
				// n.register(msg)
				ev := msg.Data.(*event.CustomRegister)
				var ext *store.ExtensionRegisterInfo
				expire, _ := strconv.Atoi(ev.Expires)
				if expire == 0 {
					// 分机签出，需要将坐席也要签出
					store.ExtensionManagerInstance.DelByCallID(ev.CallId)
				} else {
					ext = store.ExtensionManagerInstance.GetByNumber(ev.Username)

					if ext != nil {
						ext.NetworkIP = ev.NetworkIP
						ext.NetworkPort = ev.NetworkPort
						// 分机已经签入, 检查 call id是否相同
						if ext.CallID != ev.CallId {
							// 新的注册消息，需要将旧的信息清理掉，并设置新的分机
							store.ExtensionManagerInstance.DelByCallID(ev.CallId)
							store.ExtensionManagerInstance.Add(ext)
						} else {
							// 新的注册分机，需要更新信息
							ext.Refresh(expire)
						}
					} else {
						// 分机没有任何签入信息  属于新分机
						ext = store.NewExtensionRegisterInfo(ev.Username, expire, ev.CallId)
						ext.NetworkIP = ev.NetworkIP
						ext.NetworkPort = ev.NetworkPort
						// 需要增加分机是否被其他坐席签入的逻辑
						store.ExtensionManagerInstance.Add(ext)
					}
				}
			case msg := <-callstatus_chan:
				ev := msg.Data.(*event.CustomCallStatus)
				agent := agent_manager.GetByNumber(ev.Number)
				if agent != nil {
					agent.PushEvent(ev)
				}
			case msg := <-hangup_chan:
				ev := msg.Data.(*event.SessionHangup)
				agent := agent_manager.GetAgentBySession(ev.SessionId)
				if agent != nil {
					agent.PushEvent(ev)
					agent_manager.DelSession(ev.SessionId)
				}
			}
		}
	}()
	return nil
}

func (n *HttpServer) Uninit() error {
	n.uninit_flag = true
	if n.exit_sig != nil {
		close(n.exit_sig)
		n.exit_sig = nil
	}
	if n.web != nil {
		err := n.l.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

var defaultServer *HttpServer
var routers map[string]func(*gin.Context) = make(map[string]func(*gin.Context))
var registers []func(*gin.RouterGroup)
var statics []func(*gin.Engine)

func init() {
	defaultServer = &HttpServer{}
	app.GetDefaultApp().Add(0, defaultServer)
}
