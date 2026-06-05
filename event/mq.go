package event

import "sync"

// 主题类型
type Topic int

type Result struct {
	Code int
	Data any
}

// 消息结构体
type Message struct {
	Topic    Topic // 消息主题
	Data     any   // 消息内容
	Res      Result
	res_chan chan bool
	mu       sync.Mutex // 保护 res_chan
}

func (m *Message) Done(code int, data any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.res_chan == nil {
		return
	}
	m.Res.Code = code
	m.Res.Data = data
	close(m.res_chan)
	m.res_chan = nil
}

// 事件总线（订阅/注册/发布核心）
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[Topic][]chan *Message // 主题 -> 订阅者列表
}

// NewEventBus 创建总线
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[Topic][]chan *Message),
	}
}

// ------------------------------
// Subscribe 订阅（注册）消息
// ------------------------------
func (eb *EventBus) Subscribe(topic Topic) <-chan *Message {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// 创建带缓冲的通道，避免阻塞
	ch := make(chan *Message, 100)
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	return ch
}

// ------------------------------
// Publish 发布消息
// ------------------------------
func (eb *EventBus) Publish(topic Topic, data any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// 给该主题下所有订阅者发消息
	for _, ch := range eb.subscribers[topic] {
		msg := &Message{Topic: topic, Data: data, Res: Result{Code: -999}, res_chan: nil}
		go func(post_ch chan *Message, m *Message) {
			post_ch <- m
		}(ch, msg)
	}
}

// ------------------------------
// 请求消息，只支持一个模块调用
// ------------------------------
func (eb *EventBus) Request(topic Topic, data any) Result {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	chs, ok := eb.subscribers[topic]
	if !ok || len(chs) == 0 {
		return Result{
			Code: -1,
			Data: nil,
		}
	}

	ch := chs[0]
	msg := &Message{Topic: topic, Data: data, Res: Result{Code: -999}, res_chan: make(chan bool, 1)}
	go func(post_ch chan *Message, m *Message) {
		post_ch <- m
	}(ch, msg)
	<-msg.res_chan
	return msg.Res
}

var defaultBus *EventBus

func GetDefaultBus() *EventBus {
	return defaultBus
}

func init() {
	defaultBus = NewEventBus()
}
