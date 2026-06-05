package ai

import (
	"fmt"
	"testing"
)

func TestDifySay(t *testing.T) {
	vendor := NewAIDify()
	_ = vendor.Load("agent", "http://192.168.247.128/v1", "app-BlLR2NxCt0BrV0O97sl95KMy", 100)
	resp, err := vendor.Say("我要少妇，20~40的", "你是谁", 0)
	if err != nil {
		t.Errorf("Say failed: %v", err)
		return
	}
	fmt.Println("输出结果：", resp)
	t.Logf("Say resp: %v", resp)
}
