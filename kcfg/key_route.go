// key_route
package kcfg

import (
	"errors"
	"strings"
)

type keyRoute struct {
	routeStr  string
	routeList []string
	index     int
}

func (kr *keyRoute) parse(str string) {
	kr.routeStr = str
	list := strings.Split(str, ".")
	for _, v := range list {
		e := strings.TrimSpace(v)
		if len(e) != 0 {
			kr.routeList = append(kr.routeList, e)
		}
	}
}

func (kr *keyRoute) next() string {
	if kr.end() {
		panic(errors.New("元素已结束"))
	}
	ret := kr.routeList[kr.index]
	kr.index++
	return ret
}

func (kr *keyRoute) end() bool {
	return kr.index >= len(kr.routeList)
}

func NewKeyRoute() *keyRoute {
	return &keyRoute{
		index: 0,
	}
}
