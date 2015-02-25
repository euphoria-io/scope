package scope

import (
	"errors"
	"sync"
)

var (
	Cancelled = errors.New("context cancelled")
	Canceled  = Cancelled
)

type Context interface {
	Done() <-chan struct{}
	Err() error
	Cancel()
	Terminate(error)
	Fork() Context

	Get(key interface{}) interface{}
	GetOK(key interface{}) (interface{}, bool)
	Set(key, val interface{})

	WaitGroup() *sync.WaitGroup

	Breakpointer
}

type builtinKey int

const (
	bpmapKey builtinKey = iota
)

type kvmap map[interface{}]interface{}

func New() Context {
	ctx := &context{
		wg:       &sync.WaitGroup{},
		done:     make(chan struct{}),
		data:     kvmap{},
		children: map[*context]struct{}{},
	}
	ctx.Set(bpmapKey, bpmap{})
	return ctx
}

type context struct {
	wg       *sync.WaitGroup
	m        sync.RWMutex
	done     chan struct{}
	err      error
	data     kvmap
	aliased  *context
	children map[*context]struct{}
}

func (ctx *context) WaitGroup() *sync.WaitGroup { return ctx.wg }
func (ctx *context) Done() <-chan struct{}      { return ctx.done }
func (ctx *context) Err() error                 { return ctx.err }
func (ctx *context) Cancel()                    { ctx.Terminate(Cancelled) }

func (ctx *context) Terminate(err error) {
	ctx.m.Lock()
	ctx.terminate(err)
	ctx.m.Unlock()
}

func (ctx *context) terminate(err error) {
	if ctx.err == nil {
		ctx.err = err
		for child := range ctx.children {
			child.m.Lock()
			child.terminate(err)
			child.m.Unlock()
		}
		close(ctx.done)
	}
}

func (ctx *context) Fork() Context {
	ctx.m.Lock()
	defer ctx.m.Unlock()

	child := &context{
		wg:       ctx.wg,
		done:     make(chan struct{}),
		children: map[*context]struct{}{},
	}
	if ctx.aliased == nil {
		child.aliased = ctx
	} else {
		child.aliased = ctx.aliased
	}
	ctx.children[child] = struct{}{}
	return child
}

func (ctx *context) Get(key interface{}) interface{} {
	val, _ := ctx.GetOK(key)
	return val
}

func (ctx *context) GetOK(key interface{}) (interface{}, bool) {
	ctx.m.RLock()
	defer ctx.m.RUnlock()

	if ctx.aliased != nil {
		return ctx.aliased.GetOK(key)
	}

	val, ok := ctx.data[key]
	return val, ok
}

func (ctx *context) Set(key, val interface{}) {
	ctx.m.Lock()
	defer ctx.m.Unlock()

	if ctx.aliased != nil {
		ctx.data = kvmap{}
		ctx.aliased.m.RLock()
		for k, v := range ctx.aliased.data {
			ctx.data[k] = v
		}
		ctx.aliased.m.RUnlock()
		ctx.aliased = nil
	}
	ctx.data[key] = val
}

func (ctx *context) Breakpoint(scope ...interface{}) chan error {
	return ctx.Get(bpmapKey).(bpmap).get(true, scope...)
}

func (ctx *context) Check(scope ...interface{}) error {
	ch := ctx.Get(bpmapKey).(bpmap).get(false, scope...)
	if ch == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- nil:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}
