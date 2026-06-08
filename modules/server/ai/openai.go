package ai

import (
	"context"
	"errors"
	"strings"
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

// SayStream 流式调用大模型，按自然断句（标点符号）分段回调
// prefilled: 预填 assistant 开头内容，减少首 token 延迟，可为空
// onSegment: 每收到一个完整分段时回调（累积到标点符号后输出）
func (o *ai_openai) SayStream(prefilled string, prompt string, text string, timeout_ms int, onSegment func(segment string)) (string, error) {
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
		openai.SystemMessage(prompt),
	}
	messages = append(messages, o.messages...)
	messages = append(messages, openai.UserMessage(text))

	// 预填 assistant 开头，模型会接着写，大幅减少首 token 延迟
	if prefilled != "" {
		messages = append(messages, openai.AssistantMessage(prefilled))
	}

	stream := o.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    o.model,
		Messages: messages,
	})

	var fullText strings.Builder
	var segmentBuffer strings.Builder
	first := true

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}

		// 第一个 token 到达后，先回调预填内容（作为第一个分段）
		if first && prefilled != "" {
			if onSegment != nil {
				onSegment(prefilled)
			}
			fullText.WriteString(prefilled)
			first = false
		}

		fullText.WriteString(delta)
		segmentBuffer.WriteString(delta)

		// 遇到自然断句标点（中英文），将积攒的内容作为一个分段回调
		if strings.ContainsAny(delta, "。！？\n!?.;;；") {
			seg := segmentBuffer.String()
			segmentBuffer.Reset()
			if onSegment != nil {
				onSegment(seg)
			}
		}
	}

	// 输出剩余未断句的内容
	if segmentBuffer.Len() > 0 {
		seg := segmentBuffer.String()
		if onSegment != nil {
			onSegment(seg)
		}
	}

	if err := stream.Err(); err != nil {
		return "", err
	}

	result := fullText.String()
	if result == "" {
		return "", errors.New("empty response from AI stream")
	}

	// 保存到历史
	o.messages = append(o.messages, openai.AssistantMessage(result))
	if len(o.messages) >= o.max_history {
		o.messages = o.messages[1:]
	}

	return result, nil
}

func NewAIOpenAI() *ai_openai {
	return &ai_openai{}
}
