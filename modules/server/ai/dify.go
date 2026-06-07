package ai

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/safejob/dify-sdk-go"
	"github.com/safejob/dify-sdk-go/base"
	"github.com/safejob/dify-sdk-go/types"
)

type ai_dify struct {
	client         *base.Client
	app            string
	conversationId string
}

func (o *ai_dify) Load(model, url, token string, max_history int) (err error) {
	o.app = model
	o.client, err = dify.NewClient(dify.ClientConfig{
		ApiServer: url,
		ApiKey:    token,
		User:      uuid.New().String(),
	})
	return err
}

func (o *ai_dify) agentSay(_ string, text string, _ int) (string, error) {
	ctx := context.Background()
	eventCh, meta := o.client.AgentApp().Run(ctx, types.ChatRequest{
		Query:          text,
		Inputs:         map[string]any{},
		ConversationId: o.conversationId,
	}).SimplePrint()
	o.conversationId = meta.ConversationId
	var answer strings.Builder
	for msg := range eventCh {
		answer.WriteString(msg)
	}
	return answer.String(), nil
}

func (o *ai_dify) Say(prompt string, text string, timeout_ms int) (string, error) {
	switch o.app {
	case "agent":
		return o.agentSay(prompt, text, timeout_ms)
	}
	return "", nil
}

func NewAIDify() *ai_dify {
	return &ai_dify{}
}
