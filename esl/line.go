package esl

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	EslLineType_Null     = 1
	EslLineType_Str      = 2
	EslLineType_KeyValue = 3
)

func IsNullChar(c byte) bool {
	return c == '\n' || c == '\r' || c == '\t' || c == ' '
}

func NewLine() *Line {
	return &Line{
		line_type: EslLineType_Null,
	}
}

type Line struct {
	ok        bool
	line      string
	key       string
	value     string
	line_type int
}

func (l *Line) IsOK() bool {
	return l.ok
}

func (l *Line) IsNull() bool {
	return l.ok && len(l.line) == 0
}

func (l *Line) Parse(buf []byte) int {
	pos := 0
	if len(buf) <= 0 {
		return 0
	}
	for pos < len(buf) && buf[pos] != '\n' {
		pos++
	}
	if pos == 0 {
		l.ok = true
		l.parse()
		return 1
	}
	l.line = l.line + string(buf[0:pos])
	if pos < len(buf) && buf[pos] == '\n' {
		l.ok = true
		l.parse()
		pos++
	}
	return pos
}

func (l *Line) GetKey() string {
	return l.key
}

func (l *Line) GetValue() string {
	return l.value
}

func (l *Line) GetLine() string {
	return l.line
}

func (l *Line) SetLine(key, value string) {
	l.key = key
	l.value = value
	l.line = key + ":" + value
	l.line_type = EslLineType_KeyValue
	l.ok = true
}

func (l *Line) SetLineStr(str string) {
	if str == "\n" {
		l.ok = true
		l.line_type = EslLineType_Str
		l.line = ""
		return
	}
	l.line = str
	l.line_type = EslLineType_Str
	l.ok = true
}

func (l *Line) Debug() string {
	switch l.line_type {
	case EslLineType_Str:
		return fmt.Sprintf("<line><%v>\n", l.line)
	case EslLineType_KeyValue:
		return fmt.Sprintf("<key-value><%v><%v>\n", l.key, l.value)
	default:
		return "<end>\n"
	}
}

func (l *Line) Dump() string {
	if l.line_type == EslLineType_KeyValue {
		return l.key + ": " + l.value + "\n"
	}
	return l.line + "\n"
}

func (l *Line) GetLineSize() int {
	return len(l.line)
}

func (l *Line) parse() {
	l.line_type = EslLineType_Null
	if !l.IsOK() || l.IsNull() {
		return
	}
	pos := strings.Index(l.line, ":")
	if pos == -1 {
		l.line_type = EslLineType_Str
	} else {
		l.line_type = EslLineType_KeyValue
		l.key = l.line[0:pos]
		pos++
		l.value = l.line[pos:]

		for len(l.value) > 0 && IsNullChar(l.value[len(l.value)-1]) {
			l.value = l.value[:len(l.value)-1]
		}
		for len(l.value) > 0 && IsNullChar(l.value[0]) {
			l.value = l.value[1:]
		}
		if len(l.value) > 0 {
			r, e := url.PathUnescape(l.value)
			if e == nil {
				l.value = r
			}
		}
	}
}
