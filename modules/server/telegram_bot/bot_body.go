package telegram_bot

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/pbx"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/go-telegram/bot"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

const (
	IDLE_TIMEOUT = 10 * time.Minute
)

const (
	EVENT_CALLBACK_CLOSE = 1
	EVENT_CALLBACK_CALL  = 2
	EVENT_CALLBACK_TEST  = 3
)

type EventCallback func(int, any) error
type CloseCallback func()

type ButtonEvent struct {
	C chan any
}

func (p *ButtonEvent) Wait(timeout time.Duration) any {
	select {
	case v, ok := <-p.C:
		if ok {
			return v
		}
		return nil
	case <-time.After(timeout):
		return nil
	}
}

func (p *ButtonEvent) Notify(data any) {
	p.C <- data
}

func (p *ButtonEvent) Close() {
	close(p.C)
}
func NewButtonEvent() *ButtonEvent {
	return &ButtonEvent{
		C: make(chan any, 1),
	}
}

// 如果 1 小时没有消息，则关闭
type BotBody struct {
	ctx       context.Context
	engine    *LuaEngine
	b         *tgbot.Bot
	timer     *time.Timer
	callback  CloseCallback
	cfg       TelegramBotConfig
	chat_id   int64
	idx       atomic.Int64
	bt_events map[int64]*ButtonEvent
	bt_mu     sync.Mutex
	user      *tgmodels.User
	me        *tgmodels.User
}

func (p *BotBody) Start() {
	go p.engine.Run()
}

func (p *BotBody) OnHangup() {
	p.engine.OnSessionMessage(map[string]any{
		"event": "hangup",
	})
}

func (p *BotBody) OnAnswer() {
	p.engine.OnSessionMessage(map[string]any{
		"event": "answer",
	})
}

func (p *BotBody) OnDtmf(dtmf string) {
	p.engine.OnSessionMessage(map[string]any{
		"event": "dtmf",
		"data":  dtmf,
	})
}
func (p *BotBody) OnMessage(update *tgmodels.Update) {
	if p.timer != nil {
		p.timer.Reset(IDLE_TIMEOUT)
	} else {
		p.timer = time.AfterFunc(IDLE_TIMEOUT, p.Close)
	}
	p.engine.OnMessage(update)
}

func (p *BotBody) OnCallNumber(number string) error {
	var displayNumber string
	user, err := app.GetDefaultApp().GetStoreEngine().QueryTeleGramUser(store.TeleGramUser{Username: p.user.Username})
	if err != nil || user.Username == "" {
		return fmt.Errorf("%v not found", p.user.Username)
	}
	if user.IsExpired() {
		return fmt.Errorf("%v expired", p.user.Username)
	}
	var selectedExt *store.ExtensionRegisterInfo
	exts := store.ExtensionManagerInstance.GetAllOnlineByTenant(user.TenantId)
	if p.me.Username == p.user.Username {
		// 如果是自己可以使用所有号码
		if len(exts) == 0 {
			return fmt.Errorf("no ext found")
		}
		selectedExt = exts[rand.Intn(len(exts))]
		defaultLogger.Info(ThisModule, "random select ext: %v", selectedExt.Number)
		displayNumber = selectedExt.Number
	} else {
		if user.BindNumbers == "*" {
			if len(exts) == 0 {
				return fmt.Errorf("no ext found")
			}
			selectedExt = exts[rand.Intn(len(exts))]
			defaultLogger.Info(ThisModule, "random select ext: %v", selectedExt.Number)
			displayNumber = selectedExt.Number
		} else {
			numbers := strings.Split(user.BindNumbers, ";")
			if len(numbers) == 0 {
				return fmt.Errorf("no ext found")
			}
			validExts := make([]*store.ExtensionRegisterInfo, 0)
			for _, number := range numbers {
				for _, ext := range exts {
					if ext.Number == number && ext.IsValid() {
						validExts = append(validExts, ext)
					}
				}
			}
			if len(validExts) == 0 {
				return fmt.Errorf("no ext found")
			}

			selectedExt = validExts[rand.Intn(len(validExts))]
			defaultLogger.Info(ThisModule, "random select ext: %v", selectedExt.Number)
			displayNumber = selectedExt.Number
		}
	}
	uid := uuid.NewString()
	ivrId := p.cfg.ToIvrId
	if user.IvrId != "" {
		ivrId = user.IvrId
	}
	req_obj := pbx.Originate{
		TaskID:        uid,
		TenantID:      user.TenantId,
		Number:        number,
		DisplayNumber: displayNumber,
		Type:          "to_ivr",
		IvrID:         ivrId,
		Timeout:       int(p.cfg.Timeout),
		OriginateType: "ims",
		OriginateArg:  fmt.Sprintf("%v:%v", selectedExt.NetworkIP, selectedExt.NetworkPort),
	}
	globalBotManager.Set(uid, p)
	err = pbx.OriginateToIvr(req_obj)
	if err != nil {
		defaultLogger.Info(ThisModule, "<%v> Originate %v %v Failed, reason:%v", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number, err.Error())
		globalBotManager.Delete(uid)
		return err
	}
	defaultLogger.Info(ThisModule, "<%v> Originate %v %v Success", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number)
	return nil
}

func (p *BotBody) OnTestNumber(number string) error {
	user, err := app.GetDefaultApp().GetStoreEngine().QueryTeleGramUser(store.TeleGramUser{Username: p.user.Username})
	if err != nil || user.Username == "" {
		return fmt.Errorf("%v not found", p.user.Username)
	}
	if user.IsExpired() {
		return fmt.Errorf("%v expired", p.user.Username)
	}
	var selectedExt *store.ExtensionRegisterInfo
	exts := store.ExtensionManagerInstance.GetAllOnlineByTenant(user.TenantId)
	if p.me.Username == p.user.Username {
		if len(exts) == 0 {
			return fmt.Errorf("no ext found")
		}
		selectedExt = exts[rand.Intn(len(exts))]
		defaultLogger.Info(ThisModule, "OnTestNumber random select ext: %v", selectedExt.Number)
	} else {
		if user.BindNumbers == "*" {
			if len(exts) == 0 {
				return fmt.Errorf("no ext found")
			}
			selectedExt = exts[rand.Intn(len(exts))]
			defaultLogger.Info(ThisModule, "OnTestNumber random select ext: %v", selectedExt.Number)
		} else {
			numbers := strings.Split(user.BindNumbers, ";")
			if len(numbers) == 0 {
				return fmt.Errorf("no ext found")
			}
			validExts := make([]*store.ExtensionRegisterInfo, 0)
			for _, number := range numbers {
				for _, ext := range exts {
					if ext.Number == number && ext.IsValid() {
						validExts = append(validExts, ext)
					}
				}
			}
			if len(validExts) == 0 {
				return fmt.Errorf("no ext found")
			}
			selectedExt = validExts[rand.Intn(len(validExts))]
			defaultLogger.Info(ThisModule, "OnTestNumber random select ext: %v", selectedExt.Number)
		}
	}
	uid := uuid.NewString()
	ivrId := "test"
	if user.IvrId != "" {
		ivrId = user.IvrId
	}
	req_obj := pbx.Originate{
		TaskID:        uid,
		TenantID:      user.TenantId,
		Number:        number,
		DisplayNumber: selectedExt.Number,
		Type:          "to_ivr",
		IvrID:         ivrId,
		Timeout:       int(p.cfg.Timeout),
		OriginateType: "ims",
		OriginateArg:  fmt.Sprintf("%v:%v", selectedExt.NetworkIP, selectedExt.NetworkPort),
	}
	globalBotManager.Set(uid, p)
	err = pbx.OriginateToIvr(req_obj)
	if err != nil {
		defaultLogger.Info(ThisModule, "<%v> Originate %v %v Failed, reason:%v", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number, err.Error())
		globalBotManager.Delete(uid)
		return err
	}
	defaultLogger.Info(ThisModule, "<%v> Originate %v %v Success", req_obj.TaskID, req_obj.DisplayNumber, req_obj.Number)
	return nil
}

func (p *BotBody) OnEventCallback(event int, data any) error {
	switch event {
	case EVENT_CALLBACK_CLOSE:
		p.Close()
	case EVENT_CALLBACK_CALL:
		// p.b.SendMessage(p.cfg.ToIvrId, data.(string))
		return p.OnCallNumber(data.(string))
	case EVENT_CALLBACK_TEST:
		return p.OnTestNumber(data.(string))
	}
	return nil
}

func (p *BotBody) addBtEvent(idx int64, et *ButtonEvent) {
	p.bt_mu.Lock()
	defer p.bt_mu.Unlock()
	p.bt_events[idx] = et
}

func (p *BotBody) getBtEvent(idx int64) *ButtonEvent {
	p.bt_mu.Lock()
	defer p.bt_mu.Unlock()
	return p.bt_events[idx]
}

func (p *BotBody) delBtEvent(idx int64) {
	p.bt_mu.Lock()
	defer p.bt_mu.Unlock()
	delete(p.bt_events, idx)
}

func (p *BotBody) OnButtonCallback(data string) {
	arr := strings.Split(data, "_")
	if len(arr) < 4 {
		return
	}
	if arr[1] != "check" {
		return
	}
	v2, err := strconv.ParseInt(arr[2], 10, 64)
	if err != nil {
		return
	}
	v3, err := strconv.Atoi(arr[3])
	if err != nil {
		return
	}
	bt := p.getBtEvent(v2)
	if bt == nil {
		return
	}
	bt.Notify(v3)
}

func (p *BotBody) OnCheck() int {
	idx := p.idx.Load()
	p.idx.Add(1)
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "👍", CallbackData: fmt.Sprintf("button_check_%v_0", idx), Style: "success"},
				{Text: "👎", CallbackData: fmt.Sprintf("button_check_%v_1", idx), Style: "primary"},
			},
		},
	}

	et := NewButtonEvent()
	p.addBtEvent(idx, et)
	defer p.delBtEvent(idx)
	_, _ = p.b.SendMessage(p.ctx, &bot.SendMessageParams{
		ChatID:      p.chat_id,
		Text:        "Please check code ...",
		ReplyMarkup: kb,
	})
	v := et.Wait(time.Second * 60)
	defaultLogger.Info(ThisModule, "OnCheck %v", v)
	if v == nil {
		return -1
	}
	return v.(int)
}

func (p *BotBody) OnEnsure() {
	p.engine.OnSessionMessage(map[string]any{
		"event": "ensure",
	})
}

func (p *BotBody) Close() {
	if p.callback != nil {
		p.callback()
		p.callback = nil
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	if p.engine != nil {
		p.engine.Close()
	}
}

func NewBotBody(cfg TelegramBotConfig, ctx context.Context, b *tgbot.Bot, me *tgmodels.User, update *tgmodels.Update, callback CloseCallback) *BotBody {
	body := &BotBody{
		ctx:       ctx,
		b:         b,
		callback:  callback,
		cfg:       cfg,
		chat_id:   update.Message.Chat.ID,
		bt_events: make(map[int64]*ButtonEvent),
		user:      update.Message.From,
		me:        me,
	}
	body.engine = NewLuaEngine(ctx, b, update, cfg.BindScript, body.OnEventCallback)
	return body
}

type BotBodyManager struct {
	mu sync.RWMutex
	b  map[string]*BotBody
}

func (p *BotBodyManager) Get(id string) *BotBody {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.b[id]
}

func (p *BotBodyManager) Set(id string, body *BotBody) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.b[id] = body
}

func (p *BotBodyManager) GetAndDelete(id string) *BotBody {
	p.mu.Lock()
	defer p.mu.Unlock()
	body := p.b[id]
	delete(p.b, id)
	return body
}
func (p *BotBodyManager) Delete(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.b, id)
}

var globalBotManager *BotBodyManager = &BotBodyManager{
	b: make(map[string]*BotBody),
}
