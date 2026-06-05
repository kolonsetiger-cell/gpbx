package esl

func NewProto(callback msg_callback) *Proto {
	return &Proto{
		packet:   NewPackage(),
		callback: callback,
	}
}

type msg_callback interface {
	on_package(*Package)
}

type Proto struct {
	packet   *Package
	callback msg_callback
}

func (p *Proto) Parse(buf []byte) bool {
	pos := 0
	for pos < len(buf) {
		r := p.packet.Parse(buf[pos:])
		if r <= 0 {
			break
		}
		if p.packet.IsOK() {
			if p.callback != nil {
				p.callback.on_package(p.packet)
			}
			p.packet = NewPackage()
		}
		pos += r
	}

	return pos == len(buf)
}
