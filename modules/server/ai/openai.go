package ai

import (
	"context"
	"errors"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// type ai_func struct {
// 	Name   string
// 	Desc   string
// 	Params map[string]any
// }

type ai_openai struct {
	url         string
	token       string
	model       string
	client      openai.Client
	max_history int
	messages    []openai.ChatCompletionMessageParamUnion
}

func (o *ai_openai) Load(model, url, token string, max_history int) error {
	o.model = model
	o.url = url
	o.token = token
	o.client = openai.NewClient(option.WithBaseURL(o.url), option.WithAPIKey(o.token))
	o.max_history = max_history
	return nil
}

func (o *ai_openai) Say(prompt string, text string, timeout_ms int) (string, error) {
	if timeout_ms < 0 {
		timeout_ms = 10000
	}
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout_ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout_ms)*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(
			prompt,
		),
	}
	messages = append(messages, o.messages...)
	messages = append(messages, openai.UserMessage(text))
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    o.model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no choices returned from AI")
	}
	o.messages = append(o.messages, openai.AssistantMessage(resp.Choices[0].Message.Content))
	if len(o.messages) >= o.max_history {
		o.messages = o.messages[1:]
	}
	return resp.Choices[0].Message.Content, nil
}

func NewAIOpenAI() *ai_openai {
	return &ai_openai{}
}
