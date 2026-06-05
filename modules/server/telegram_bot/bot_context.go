package telegram_bot

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/go-telegram/bot"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	tgmodels "github.com/go-telegram/bot/models"
	"golang.org/x/net/proxy"
)

type BotContext struct {
	bot     *tgbot.Bot
	cfg     TelegramBotConfig
	ctx     context.Context
	cancel  context.CancelFunc
	me      *tgmodels.User
	bot_map map[int64]*BotBody
	bot_mu  sync.Mutex
}

func (m *BotContext) callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	body := m.getBotBody(update.CallbackQuery.Message.Message.Chat.ID)
	if body == nil {
		defaultLogger.Error(ThisModule, "callbackHandler: bot body not found")
		return
	}
	body.OnButtonCallback(update.CallbackQuery.Data)
}

func (m *BotContext) defaultHandler(ctx context.Context, b *tgbot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}

	from := update.Message.From
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	defaultLogger.Info(ThisModule, "Received message from %s (ID:%d) in chat %d: %s",
		from.Username, from.ID, chatID, text)
	body := m.getBotBody(chatID)
	if body == nil {
		cfg := m.cfg
		user, err := app.GetDefaultApp().GetStoreEngine().QueryTeleGramUser(store.TeleGramUser{Username: from.Username})
		if err != nil || user.Username == "" {
			defaultLogger.Error(ThisModule, "defaultHandler: get user failed, err:%v", err)
			return
		}
		cfg.BindScript = user.BindScript
		body = NewBotBody(cfg, ctx, b, m.me, update, func() {
			m.delBotBody(chatID)
		})
		if body == nil {
			defaultLogger.Error(ThisModule, "defaultHandler: new bot body failed")
			return
		}
		m.addBotBody(chatID, body)
		body.Start()
	}
	body.OnMessage(update)
}

func (p *BotContext) Start(cfg TelegramBotConfig) error {
	p.cfg = cfg
	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(p.defaultHandler),
		tgbot.WithCheckInitTimeout(time.Minute),
		tgbot.WithSkipGetMe(),
		bot.WithCallbackQueryDataHandler("button", bot.MatchTypePrefix, p.callbackHandler),
	}
	if p.cfg.Proxy != "" {
		proxyURL, err := url.Parse(p.cfg.Proxy)
		if err != nil {
			defaultLogger.Error(ThisModule, "Invalid proxy URL '%s': %v", p.cfg.Proxy, err)
		} else {
			transport := &http.Transport{}

			switch proxyURL.Scheme {
			case "socks5", "socks5h":
				// SOCKS5 代理 (QuickQ 默认使用此类型)
				auth := proxy.Auth{}
				if proxyURL.User != nil {
					pwd, _ := proxyURL.User.Password()
					auth.User = proxyURL.User.Username()
					auth.Password = pwd
				}
				dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, &auth, proxy.Direct)
				if err != nil {
					defaultLogger.Error(ThisModule, "Failed to create SOCKS5 dialer: %v", err)
				} else {
					transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
						return dialer.Dial(network, addr)
					}
					defaultLogger.Info(ThisModule, "TelegramBot using SOCKS5 proxy: %s", proxyURL.Host)
				}

			default:
				// HTTP/HTTPS 代理
				transport.Proxy = http.ProxyURL(proxyURL)
				defaultLogger.Info(ThisModule, "TelegramBot using HTTP proxy: %s", proxyURL.Host)
			}

			opts = append(opts, tgbot.WithHTTPClient(time.Minute, &http.Client{
				Transport: transport,
				Timeout:   time.Minute,
			}))
		}
	}
	bot, err := tgbot.New(p.cfg.Token, opts...)
	if err != nil {
		defaultLogger.Error(ThisModule, "Failed to create bot: %v", err)
		return err
	}
	p.bot = bot
	me, err := p.bot.GetMe(p.ctx)
	if err == nil {
		defaultLogger.Info(ThisModule, "TelegramBot started, bot username: %s", me.Username)
	} else {
		defaultLogger.Info(ThisModule, "TelegramBot getme failed, err:%v", err)
		return err
	}
	p.me = me
	go func() {
		p.bot.Start(p.ctx)
	}()
	return nil
}

func (p *BotContext) Stop() {
	defaultLogger.Info(ThisModule, "TelegramBot stoped, bot username: %s", p.me.Username)
	p.cancel()
}

func (m *BotContext) getBotBody(chat_id int64) *BotBody {
	m.bot_mu.Lock()
	defer m.bot_mu.Unlock()
	bot, ok := m.bot_map[chat_id]
	if !ok {
		return nil
	}
	return bot
}

func (m *BotContext) addBotBody(chat_id int64, body *BotBody) {
	m.bot_mu.Lock()
	defer m.bot_mu.Unlock()
	m.bot_map[chat_id] = body
}
func (m *BotContext) delBotBody(chat_id int64) {
	m.bot_mu.Lock()
	defer m.bot_mu.Unlock()
	delete(m.bot_map, chat_id)
}

func NewBotContext() *BotContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &BotContext{
		ctx:     ctx,
		cancel:  cancel,
		bot_map: make(map[int64]*BotBody),
	}
}
