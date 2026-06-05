package callcenter_api

import (
	"fmt"
	"net"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/log"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const ThisModule = "HttpServer"

var defaultLogger log.Logger

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type HttpServer struct {
	web         *gin.Engine
	l           net.Listener
	uninit_flag bool
}

func (n *HttpServer) SetLogger(logger log.Logger) {
	defaultLogger = logger
}

func (n *HttpServer) Init() error {
	n.web = gin.Default()

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

	host := app.GetDefaultApp().GetCfg().Child("http.host").GetString()
	port := app.GetDefaultApp().GetCfg().Child("http.api_port").GetInt()
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

func init() {
	defaultServer = &HttpServer{}
	app.GetDefaultApp().Add(0, defaultServer)
}
