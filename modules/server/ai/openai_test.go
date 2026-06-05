package ai

import (
	"fmt"
	"testing"
)

func TestSay(t *testing.T) {
	vendor := NewAIOpenAI()
	_ = vendor.Load("qwen2.5-7b-instruct", "http://192.168.0.108:1234/v1", "sk-", 100)
	resp, err := vendor.Say("你是一个专业的翻译,输出为 json", "你好", 0)
	if err != nil {
		t.Errorf("Say failed: %v", err)
		return
	}
	fmt.Println(resp)
	t.Logf("Say resp: %v", resp)
}
