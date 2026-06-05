package kcfg

import (
	"reflect"
	"testing"
)

func assertEqual(t *testing.T, left, right interface{}, err string) {
	if reflect.TypeOf(left).Kind() != reflect.TypeOf(right).Kind() {
		t.Error(err)
	}
	switch reflect.TypeOf(left).Kind() {
	case reflect.Bool:
		if reflect.ValueOf(left).Bool() != reflect.ValueOf(right).Bool() {
			t.Error(err)
		}
	case reflect.String:
		if reflect.ValueOf(left).String() != reflect.ValueOf(right).String() {
			t.Error(err)
		}
	}
}

func TestNewNode(t *testing.T) {
	node := NewNode("root")
	child := NewNode("child")
	child.key = "child"
	node.AddChild(child)
	assertEqual(t, "child", node.Childs("child")[0].key, "not child")
}
