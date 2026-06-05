package esl

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/log"
)

const ThisModule string = "ESL"
const RECV_CACHE_SIZE = 4096
const (
	ESLClientState_Null       = 0
	ESLClientState_Connecting = 1
	ESLClientState_Connected  = 2
	ESLClientState_Auth       = 3
	ESLClientState_SignIn     = 4
	ESLClientState_Success    = 5
	ESLClientState_SubEvent   = 6
)

type FSConfig struct {
	Addr                      string
	User                      string
	Pass                      string
	Reconnect_Interval_Second int // 重连间隔，单位秒，<=0 表示不重连
}

type EventCallback func(*Client, int)
type PackageCallback func(string, *Package)

func NewClient() *Client {
	c := &Client{
		state:       ESLClientState_Null,
		sub_msg:     make(map[string]PackageCallback),
		cus_sub_msg: make(map[string]PackageCallback),
		logger:      &DefaultLogger{},
	}
	return c
}

type DefaultLogger struct {
}

func (l *DefaultLogger) Debug(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

func (l *DefaultLogger) Info(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

func (l *DefaultLogger) Warn(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

func (l *DefaultLogger) Error(module string, format string, arg ...any) {
	fmt.Printf(format+"\n", arg...)
}

type Client struct {
	state            int
	config           FSConfig
	sub_msg          map[string]PackageCallback
	cus_sub_msg      map[string]PackageCallback
	all_msg          PackageCallback
	io               net.Conn
	proto            *Proto
	logger           log.Logger
	event_notify     EventCallback
	enable_reconnect bool
	reconnect_timer  *time.Timer
	mu               sync.RWMutex
}

func (c *Client) set_state(state int) {
	c.state = state
	if c.event_notify != nil {
		c.event_notify(c, c.state)
	}
}

func (c *Client) format_event(event string) string {
	return fmt.Sprintf("event plain %v\n\n", event)
}

func (c *Client) on_package(pck *Package) {
	go func(pck *Package) {
		event_name := pck.GetBody("Event-Name")
		if event_name == "CUSTOM" {
			cus_event_name := pck.GetBody("Event-Subclass")
			c.logger.Debug(ThisModule, "Recv %v", cus_event_name)
			if c.all_msg != nil {
				c.all_msg(cus_event_name, pck)
			} else {
				cb, ok := c.cus_sub_msg[cus_event_name]
				if ok && cb != nil {
					cb(cus_event_name, pck)
				}
			}
		} else if len(event_name) != 0 {
			if c.all_msg != nil {
				c.all_msg(event_name, pck)
			} else {
				cb, ok := c.sub_msg[event_name]
				if ok && cb != nil {
					c.logger.Debug(ThisModule, "Pub %v", event_name)
					cb(event_name, pck)
				} else {
					c.logger.Debug(ThisModule, "Discard %v", event_name)
				}
			}
		}
	}(pck)
}

func (c *Client) SetLogger(logger log.Logger) {
	c.logger = logger
}

func (c *Client) OnEventCallback(callback EventCallback) *Client {
	c.event_notify = callback
	return c
}

func (c *Client) On(key string, callback PackageCallback) *Client {
	if key == "all" {
		c.all_msg = callback
	} else {
		c.sub_msg[key] = callback
	}
	return c
}

func (c *Client) OnCustom(key string, callback PackageCallback) *Client {
	c.cus_sub_msg[key] = callback
	return c
}

func (c *Client) Connect(config FSConfig) error {
	c.config = config
	if config.Reconnect_Interval_Second > 0 {
		c.enable_reconnect = true
	}
	c.set_state(ESLClientState_Connecting)
	conn, err := net.DialTimeout("tcp", config.Addr, 2*time.Second)
	if err != nil {
		// 开启重连
		c.set_state(ESLClientState_Null)
		c.start_reconnect()
		return err
	}
	c.io = conn
	c.proto = NewProto(c)
	c.set_state(ESLClientState_Connected)
	c.set_state(ESLClientState_Auth)
	// start send auth
	// 开始接收 auth package
	pck := NewPackage()
	buf := make([]byte, RECV_CACHE_SIZE)

	//  1. 整理出所有要订阅的事件，放到一个列表里
	sub_events := []string{}
	if c.all_msg != nil {
		c.io.Write([]byte(c.format_event("all")))
		sub_events = append(sub_events, c.format_event("all"))
	} else {
		for k := range c.cus_sub_msg {
			sub_events = append(sub_events, c.format_event("CUSTOM "+k))
		}
		for k := range c.sub_msg {
			sub_events = append(sub_events, c.format_event(k))
		}
	}

	check_timer := time.NewTimer(time.Second * time.Duration(3))
	go func() {
		<-check_timer.C
		c.logger.Error(ThisModule, "ESL Auth Timeout")
		check_timer.Stop()

		if c.io != nil {
			c.io.Close()
		}
	}()
	for c.state != ESLClientState_Success {
		n, err := c.io.Read(buf)
		if err != nil {
			check_timer.Stop()
			c.io.Close()
			c.set_state(ESLClientState_Null)
			c.start_reconnect()
			return err
		}
		pck.Parse(buf[:n])
		if pck.IsOK() {
			switch c.state {
			case ESLClientState_Auth:
				if pck.Get("Content-Type") == "auth/request" {
					c.set_state(ESLClientState_SignIn)
					if len(c.config.User) == 0 {
						fmt.Fprintf(c.io, "auth %v\n\n", c.config.Pass)
					} else {
						fmt.Fprintf(c.io, "userauth %v:%v\n\n", c.config.User, c.config.Pass)
					}
				}
			case ESLClientState_SignIn:
				if pck.Get("Content-Type") == "command/reply" {
					if pck.Get("Reply-Text") == "+OK accepted" {
						// 开始发送事件订阅
						if len(sub_events) == 0 {
							c.set_state(ESLClientState_Success)
							c.logger.Info(ThisModule, "Connect FS Success, But hasn't any event")
						} else {
							c.set_state(ESLClientState_SubEvent)
							event := sub_events[0]
							sub_events = sub_events[1:]
							c.io.Write([]byte(event))
						}
					} else if pck.Get("Reply-Text") == "" {
						c.logger.Warn(ThisModule, "Ignore Msg:%v", pck.Debug())
					} else {
						check_timer.Stop()
						c.io.Close()
						c.set_state(EslLineType_Null)
						// 开始启动重连机制
						c.logger.Warn(ThisModule, "Auth Failed, Will Reconnect")
						c.start_reconnect()
						return errors.New("Auth Failed")
					}
				}
			case ESLClientState_SubEvent:
				// 开始订阅消息
				c.logger.Debug(ThisModule, "Sub Response:%v", pck.Dump())
				if pck.Get("Content-Type") == "command/reply" {
					if strings.Index(pck.Get("Reply-Text"), "+OK") == 0 {
						// 订阅完成一次事件
						if len(sub_events) == 0 {
							c.set_state(ESLClientState_Success)
							c.logger.Info(ThisModule, "Connect FS Success")
						} else {
							c.set_state(ESLClientState_SubEvent)
							event := sub_events[0]
							sub_events = sub_events[1:]
							c.io.Write([]byte(event))
						}
					}
				}
			}
			pck = NewPackage()
		}
	}
	check_timer.Stop()
	go c.recv_msg()
	return nil
}

func (c *Client) Disconnect() {
	c.enable_reconnect = false
	if c.reconnect_timer != nil {
		c.reconnect_timer.Stop()
	}
	if c.io != nil {
		c.io.Close()
	}
}

func (c *Client) start_reconnect() {
	if !c.enable_reconnect {
		return
	}
	c.reconnect_timer = time.NewTimer(time.Second * time.Duration(c.config.Reconnect_Interval_Second))
	go func() {
		<-c.reconnect_timer.C
		c.reconnect_timer.Stop()

		if c.enable_reconnect {
			c.logger.Debug(ThisModule, "Start Reconnecting")
			c.Connect(c.config)
		}
	}()
}

func (c *Client) recv_msg() {
	buf := make([]byte, RECV_CACHE_SIZE)
	c.logger.Debug(ThisModule, "Esl Recv Start")
	for {
		n, err := c.io.Read(buf)
		if err != nil {
			c.io = nil
			c.set_state(ESLClientState_Null)
			c.start_reconnect()
			break
		}
		// c.logger.Debug(ThisModule, "%v", string(buf[:n]))
		c.proto.Parse(buf[:n])
	}
	c.proto = nil
	c.logger.Debug(ThisModule, "Esl Recv Stop")
}

func (c *Client) SendAPP(cmd, arg, uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != ESLClientState_Success {
		c.logger.Error(ThisModule, "Send APP Failed, Session not health <%v:%v:%v>",
			uuid, cmd, arg)
		return errors.New("ESL Not Idle")
	}
	var cmd_str strings.Builder
	cmd_str.WriteString("sendmsg")
	if len(uuid) != 0 {
		cmd_str.WriteString(" ")
		cmd_str.WriteString(uuid)
	}
	cmd_str.WriteString("\ncall-command: execute\n")
	if len(cmd) != 0 {
		cmd_str.WriteString("execute-app-name: ")
		cmd_str.WriteString(cmd)
		cmd_str.WriteString("\n")
	}
	if len(arg) != 0 {
		cmd_str.WriteString("execute-app-arg: ")
		cmd_str.WriteString(arg)
		cmd_str.WriteString("\n")
	}
	cmd_str.WriteString("event-lock: true\n")
	cmd_str.WriteString("async: true\n")
	cmd_str.WriteString("\n")
	_, err := c.io.Write([]byte(cmd_str.String()))
	if err != nil {
		c.logger.Error(ThisModule, "Send APP Failed, %v <%v:%v:%v>",
			err.Error(), uuid, cmd, arg)
		return err
	}
	c.logger.Info(ThisModule, "Send APP Success <%v:%v:%v>",
		uuid, cmd, arg)
	return nil
}

func (c *Client) SendAPI(cmd, arg, uuid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != ESLClientState_Success {
		c.logger.Error(ThisModule, "Send API Failed, Session not health <%v:%v:%v>",
			uuid, cmd, arg)
		return errors.New("ESL Not Idle")
	}
	var cmd_str strings.Builder
	cmd_str.WriteString("bgapi ")
	cmd_str.WriteString(cmd)
	if len(arg) != 0 {
		cmd_str.WriteString(" ")
		cmd_str.WriteString(arg)
	}
	cmd_str.WriteString("\n")
	if len(uuid) != 0 {
		cmd_str.WriteString("Job-UUID: ")
		cmd_str.WriteString(uuid)
		cmd_str.WriteString("\n")
	}
	cmd_str.WriteString("\n")
	_, err := c.io.Write([]byte(cmd_str.String()))
	if err != nil {
		c.logger.Error(ThisModule, "Send API Failed, %v <%v:%v:%v>",
			err.Error(), uuid, cmd, arg)
		return err
	}
	c.logger.Info(ThisModule, "Send API Success <%v:%v:%v>",
		uuid, cmd, arg)
	return nil
}
