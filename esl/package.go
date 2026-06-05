package esl

import (
	"math"
	"strconv"
	"strings"
)

const (
	EPacketState_ParseHeader = 1
	EPacketState_ParseBody   = 2
	EPacketState_Completed   = 3
)

func NewPackage() *Package {
	return &Package{
		state:    EPacketState_ParseHeader,
		body_map: make(map[string]*Line),
	}
}

type Package struct {
	headers           []*Line
	body              []*Line
	body_reply        string
	state             int
	body_map          map[string]*Line
	body_length       int
	body_pos          int
	body_reply_length int
}

func (p *Package) IsOK() bool {
	return p.state == EPacketState_Completed
}

func (p *Package) Parse(buf []byte) int {
	if len(buf) == 0 {
		return 0
	}
	pos := 0
	if p.state == EPacketState_ParseHeader {
		r := p.parseHeader(buf)
		if r <= 0 {
			return 0
		}
		pos += r

		content_length := p.GetAsInt("Content-Length")
		if p.IsHeadEnd() {
			if content_length <= 0 {
				p.state = EPacketState_Completed
				return pos
			}
			p.body_length = content_length
			p.body_pos = 0
			p.state = EPacketState_ParseBody
		}
	}

	if p.state == EPacketState_ParseBody && pos < len(buf) {
		pos += p.parseBody(buf[pos:])
	}
	return pos
}

func (p *Package) IsHeadEnd() bool {
	if len(p.headers) == 0 {
		return false
	}

	return p.headers[len(p.headers)-1].IsNull()
}

func (p *Package) HeadCount() int {
	return len(p.headers)
}

func (p *Package) Debug() string {
	var ret strings.Builder
	for _, l := range p.headers {
		ret.WriteString(l.Debug())
	}
	ret.WriteString("\n")
	for _, l := range p.body {
		ret.WriteString(l.Debug())
	}
	return ret.String()
}

func (p *Package) Dump() string {
	var ret strings.Builder
	for _, l := range p.headers {
		ret.WriteString(l.Dump())
	}
	for _, l := range p.body {
		ret.WriteString(l.Dump())
	}
	return ret.String()
}

func (p *Package) Get(key string) string {
	for _, l := range p.headers {
		if !l.IsOK() || l.IsNull() {
			break
		}
		if l.GetKey() == key {
			return l.GetValue()
		}
	}
	return ""
}

func (p *Package) GetAsInt(key string) int {
	v := p.Get(key)
	vi, _ := strconv.Atoi(v)
	return vi
}

func (p *Package) GetBody(key string) string {
	l, ok := p.body_map[key]
	if ok {
		return l.GetValue()
	}
	return ""
}

func (p *Package) GetBodyAsInt(key string) int {
	v := p.GetBody(key)
	vi, _ := strconv.Atoi(v)
	return vi
}

func (p *Package) HeaderBegin() *Package {
	p.headers = []*Line{}
	return p
}

func (p *Package) AddHeader(key, value string) *Package {
	l := NewLine()
	l.SetLine(key, value)
	p.headers = append(p.headers, l)
	return p
}

func (p *Package) AddHeaderStr(str string) *Package {
	l := NewLine()
	l.SetLineStr(str)
	p.headers = append(p.headers, l)
	return p
}

func (p *Package) BodyBegin(str string) *Package {
	p.body = []*Line{}
	p.body_reply = ""
	p.body_pos = 0
	p.body_reply_length = 0
	p.body_map = map[string]*Line{}
	p.body_reply = ""
	p.body_length = 0
	return p
}

func (p *Package) AddBody(key, value string) *Package {
	l := NewLine()
	l.SetLine(key, value)
	p.body = append(p.body, l)
	return p
}

func (p *Package) AddBodyStr(str string) *Package {
	l := NewLine()
	l.SetLineStr(str)
	p.body = append(p.body, l)
	return p
}

func (p *Package) SetSubBody(str string) *Package {
	p.body_reply = str
	return p
}

func (p *Package) Serialize() string {
	if len(p.body_reply) > 0 {
		p.AddBody("Content-Length", strconv.Itoa(len(p.body_reply)))
	}
	if len(p.body) > 0 {
		p.AddBodyStr("\n")
	}
	var body strings.Builder
	for _, l := range p.body {
		body.WriteString(l.Dump())
	}
	if len(body.String()) > 0 {
		p.AddHeader("Content-Length", strconv.Itoa(len(p.body_reply)+len(body.String())))
	}
	p.AddHeaderStr("\n")

	var header strings.Builder
	for _, l := range p.headers {
		header.WriteString(l.Dump())
	}
	return header.String() + body.String() + p.body_reply
}

func (p *Package) GetLine() *Line {
	if len(p.headers) == 0 || p.headers[len(p.headers)-1].IsOK() {
		line := NewLine()
		p.headers = append(p.headers, line)
	}
	return p.headers[len(p.headers)-1]
}

func (p *Package) GetBodyLine() *Line {
	if len(p.body) == 0 || p.body[len(p.body)-1].IsOK() {
		if len(p.body) > 0 && !p.body[len(p.body)-1].IsNull() {
			line := p.body[len(p.body)-1]
			p.body_map[line.GetKey()] = line
		}
		line := NewLine()
		p.body = append(p.body, line)
	}
	return p.body[len(p.body)-1]
}

func (p *Package) parseHeader(buf []byte) int {
	pos := 0
	for pos < len(buf) && !p.IsHeadEnd() {
		r := p.parseLine(buf[pos:])
		if r <= 0 {
			break
		}
		pos += r
	}
	return pos
}

func (p *Package) parseBody(buf []byte) int {
	remainSize := p.body_length - p.body_pos
	copySize := int(math.Min(float64(remainSize), float64(len(buf))))
	pos := 0
	for pos < copySize {
		if len(p.body) > 0 &&
			p.body[len(p.body)-1].IsNull() &&
			p.body_reply_length > 0 &&
			len(p.body_reply) < p.body_reply_length {
			reply_offset := int(math.Min(float64(copySize-pos), float64(p.body_reply_length)))
			p.body_reply = p.body_reply + string(buf[pos:pos+reply_offset])
			pos += reply_offset
			continue
		}
		line := p.GetBodyLine()
		r := line.Parse(buf[pos:])
		pos += r
		if line.GetKey() == "Content-Length" {
			p.body_reply_length, _ = strconv.Atoi(line.GetValue())
		}
	}
	p.body_pos += copySize
	if remainSize == copySize {
		p.state = EPacketState_Completed
		if p.body_reply_length > 0 {
			offset := len(p.body_reply)
			for offset > 0 && IsNullChar(p.body_reply[offset-1]) {
				offset--
			}
			p.body_reply = p.body_reply[0:offset]
		}
	}
	return copySize
}

func (p *Package) parseLine(buf []byte) int {
	if len(buf) == 0 {
		return 0
	}
	line := p.GetLine()
	return line.Parse(buf)
}
