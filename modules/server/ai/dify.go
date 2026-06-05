package ai

import (
	"context"

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

func (o *ai_dify) agentSay(prompt string, text string, timeout_ms int) (string, error) {
	ctx := context.Background()
	eventCh, meta := o.client.AgentApp().Run(ctx, types.ChatRequest{
		Query:          text,
		Inputs:         map[string]any{},
		ConversationId: o.conversationId,
	}).SimplePrint()
	o.conversationId = meta.ConversationId
	var answer string
	for msg := range eventCh {
		answer += msg
	}
	return answer, nil
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
