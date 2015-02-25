package scope

type Breakpointer interface {
	Breakpoint(scope ...interface{}) chan error
	Check(scope ...interface{}) error
}

type breakpoint struct {
	c chan error
}

type bpmap kvmap

func (b bpmap) get(create bool, scope ...interface{}) chan error {
	switch len(scope) {
	case 0:
		return nil
	case 1:
		if obj, ok := b[scope[0]]; ok {
			if ch, ok := obj.(chan error); ok {
				return ch
			}
			return nil
		}
		if !create {
			return nil
		}
		ch := make(chan error)
		b[scope[0]] = ch
		return ch
	default:
		if obj, ok := b[scope[0]]; ok {
			if bpm, ok := obj.(bpmap); ok {
				return bpm.get(create, scope[1:]...)
			}
			return nil
		}
		bpm := bpmap{}
		b[scope[0]] = bpm
		return bpm.get(create, scope[1:]...)
	}
}
