// assert
package kcfg

import (
	"errors"
	"reflect"
)

func AssertEqual(left, right interface{}, err string) {
	if reflect.TypeOf(left).Kind() != reflect.TypeOf(right).Kind() {
		panic(errors.New("类型不匹配"))
	}
	switch reflect.TypeOf(left).Kind() {
	case reflect.Bool:
		if reflect.ValueOf(left).Bool() != reflect.ValueOf(right).Bool() {
			panic(errors.New(err))
		}
	case reflect.String:
		if reflect.ValueOf(left).String() != reflect.ValueOf(right).String() {
			panic(errors.New(err))
		}
	}
}
