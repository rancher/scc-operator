package initializer

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testingInitializer struct {
	InitHandler
}

func (ti *testingInitializer) Reset() {
	// Reset the initialization state and reinitialize synchronization primitives
	ti.InitHandler.initialized = false
	ti.InitHandler.setupCondOnce = sync.Once{}
	ti.InitHandler.initCond = nil
}

func TestInitializedDefaultFalse(t *testing.T) {
	var ti testingInitializer
	if ti.Initialized() {
		t.Fatalf("Initialized() = true; want false before InitOnce")
	}
}

func TestInitOnceSetsInitializedAndCallsOnce(t *testing.T) {
	asserts := assert.New(t)

	var ti testingInitializer
	var calls int32
	var wg sync.WaitGroup
	concurrency := 50
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			ti.InitOnce(func() {
				// Add a small sleep to increase contention window
				atomic.AddInt32(&calls, 1)
				time.Sleep(10 * time.Millisecond)
			})
		}()
	}
	wg.Wait()
	asserts.Equal(int32(1), atomic.LoadInt32(&calls))
	asserts.True(ti.Initialized())
}

func TestWaitForInitBlocksAndUnblocks(t *testing.T) {
	var ti testingInitializer
	waiters := 10
	started := make(chan struct{}, waiters)
	done := make(chan struct{}, waiters)

	for i := 0; i < waiters; i++ {
		go func() {
			started <- struct{}{}
			ti.WaitForInit()
			done <- struct{}{}
		}()
	}

	// Ensure all waiters have started and are blocked
	for i := 0; i < waiters; i++ {
		select {
		case <-started:
			// ok
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout waiting for waiter %d to start", i)
		}
	}

	// Verify none finished yet
	select {
	case <-done:
		t.Fatalf("waiter finished before initialization; expected to be blocked")
	case <-time.After(50 * time.Millisecond):
		// expected: still blocked
	}

	// Now initialize and ensure all are released
	ti.InitOnce(func() {})

	for i := 0; i < waiters; i++ {
		select {
		case <-done:
			// ok
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timeout waiting for waiter %d to finish after init", i)
		}
	}
}

func TestWaitForInitContextTimeout(t *testing.T) {
	asserts := assert.New(t)

	var ti testingInitializer
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := ti.WaitForInitContext(ctx)
	asserts.Error(err)
	dl := time.Since(start)
	asserts.Greater(dl, 40*time.Millisecond, "WaitForInitContext returned too early: %v", dl)
}

func TestWaitForInitContextSucceeds(t *testing.T) {
	asserts := assert.New(t)

	var ti testingInitializer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Initialize after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		ti.InitOnce(func() {})
	}()
	asserts.NoError(ti.WaitForInitContext(ctx))
}

func TestWaitAfterInitializedImmediateReturn(t *testing.T) {
	var ti testingInitializer
	ti.InitOnce(func() {})
	done := make(chan struct{})
	go func() {
		ti.WaitForInit()
		close(done)
	}()
	select {
	case <-done:
		// ok, returned quickly
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("WaitForInit did not return quickly after already initialized")
	}
}
