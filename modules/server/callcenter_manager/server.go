package callcenter_manager

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/event"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const ThisModule = "HttpServer"

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

func initDefaultAdmin() {
	count := app.GetDefaultApp().GetStoreEngine().CountUser()
	if count == 0 {
		cfg := app.GetDefaultApp().GetCfg()
		defaultUsername := cfg.Child("http.admin.username").GetString()
		defaultPassword := cfg.Child("http.admin.password").GetString()
		if defaultUsername == "" {
			defaultUsername = "admin"
		}
		if defaultPassword == "" {
			defaultPassword = "admin123" // 仅当配置未设置时使用默认值
		}
		// 创建默认管理员用户
		admin := store.User{
			Username: defaultUsername,
			Password: defaultPassword, // 密码会在保存前自动加密
			Name:     "管理员",
			Roles:    "R_SUPER",
		}
		err := app.GetDefaultApp().GetStoreEngine().StoreUser(admin)
		if err != nil {
			panic(err)
		}
	}
}

func (n *HttpServer) Init() error {
	initDefaultAdmin()
	n.web = gin.Default()
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

	for r, f := range routers {
		n.web.POST(r, f)
	}
	api := n.web.Group("/api")
	for _, r := range registers {
		defaultLogger.Info(ThisModule, "Register %v", r)
		r(api)
	}
	// setupStaticFiles(n.web)
	for _, s := range statics {
		s(n.web)
	}

	host := app.GetDefaultApp().GetCfg().Child("http.host").GetString()
	port := app.GetDefaultApp().GetCfg().Child("http.manager_port").GetInt()
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
	n.l = ln
	n.uninit_flag = false
	defaultLogger.Info(ThisModule, "Http Server Listen addr <%v:%v>", host, port)
	return nil
}

func (n *HttpServer) Run() error {
	register_chan := event.GetDefaultBus().Subscribe(event.TOPIC_REGISTER)

	go func() {
		exit := false
		for !exit {
			select {
			case <-n.exit_sig:
				exit = true
			case msg := <-register_chan:
				// n.register(msg)
				ev := msg.Data.(*event.CustomRegister)
				expire, _ := strconv.Atoi(ev.Expires)
				pos := strings.Index(ev.Username, "-")
				if pos == -1 {
					continue
				}
				tenantId := ev.Username[0:pos]
				extId := ev.Username[pos+1:]
				if expire == 0 {
					_ = app.GetDefaultApp().GetStoreEngine().StoreExtension(store.Extension{
						TenantId:    tenantId,
						ExtensionId: extId,
						Status:      store.EXTENSION_STATUS_OFFLINE,
					})
				} else {
					_ = app.GetDefaultApp().GetStoreEngine().StoreExtension(store.Extension{
						TenantId:    tenantId,
						ExtensionId: extId,
						Status:      store.EXTENSION_STATUS_ONLINE,
						NetworkIP:   ev.NetworkIP,
						NetworkPort: ev.NetworkPort,
					})
				}
			}
		}
	}()
	go func() {
		err := n.web.RunListener(n.l)
		if !n.uninit_flag && err != nil {
			defaultLogger.Error(ThisModule, "Http Server err:%v", err.Error())
			panic(err)
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
