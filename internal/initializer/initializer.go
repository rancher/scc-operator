package initializer

import (
	"context"
	"sync"
)

type Initializer interface {
	InitOnce(f func())
	Initialized() bool
	WaitForInit()
	WaitForInitContext(ctx context.Context) error
}

type InitHandler struct {
	setupCondOnce sync.Once
	initCond      *sync.Cond
	initialized   bool
}

func (i *InitHandler) InitOnce(f func()) {
	i.checkInitCond()
	i.initCond.L.Lock()
	defer i.initCond.L.Unlock()
	if i.initialized {
		return
	}
	f()
	i.initialized = true
	i.initCond.Broadcast()
}

func (i *InitHandler) Initialized() bool {
	i.checkInitCond()
	i.initCond.L.Lock()
	defer i.initCond.L.Unlock()
	return i.initialized
}

func (i *InitHandler) WaitForInit() {
	i.checkInitCond()
	i.initCond.L.Lock()
	for !i.initialized {
		i.initCond.Wait()
	}
	i.initCond.L.Unlock()
}

func (i *InitHandler) WaitForInitContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		i.WaitForInit()
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (i *InitHandler) checkInitCond() {
	i.setupCondOnce.Do(func() {
		i.initCond = sync.NewCond(&sync.Mutex{})
	})
}

var _ Initializer = (*InitHandler)(nil)
