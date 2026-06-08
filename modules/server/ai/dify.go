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

// SayStream 流式调用 Dify，按自然断句分段回调
func (o *ai_dify) SayStream(prefilled string, prompt string, text string, timeout_ms int, onSegment func(segment string)) (string, error) {
	ctx := context.Background()
	eventCh, meta := o.client.AgentApp().Run(ctx, types.ChatRequest{
		Query:          text,
		Inputs:         map[string]any{},
		ConversationId: o.conversationId,
	}).SimplePrint()
	o.conversationId = meta.ConversationId

	// 先回调预填内容
	if prefilled != "" && onSegment != nil {
		onSegment(prefilled)
	}

	var fullText strings.Builder
	var segmentBuffer strings.Builder
	if prefilled != "" {
		fullText.WriteString(prefilled)
	}

	for msg := range eventCh {
		fullText.WriteString(msg)
		segmentBuffer.WriteString(msg)

		// 遇到自然断句标点，将积攒内容作为分段回调
		if strings.ContainsAny(msg, "。！？\n!?.;;；") {
			seg := segmentBuffer.String()
			segmentBuffer.Reset()
			if onSegment != nil {
				onSegment(seg)
			}
		}
	}

	// 输出剩余内容
	if segmentBuffer.Len() > 0 {
		if onSegment != nil {
			onSegment(segmentBuffer.String())
		}
	}

	return fullText.String(), nil
}

func NewAIDify() *ai_dify {
	return &ai_dify{}
}
