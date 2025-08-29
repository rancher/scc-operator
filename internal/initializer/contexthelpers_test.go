package initializer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Override globals to use testingInitializer instead of InitHandler for test control.
func init() {
	DevMode = valueInitializer[bool]{init: &testingInitializer{}}
	SystemNamespace = valueInitializer[string]{init: &testingInitializer{}}
	OperatorName = valueInitializer[string]{init: &testingInitializer{}}
}

// resetAll ensures the underlying initializers are reset for each test.
func resetAll() {
	if ti, ok := DevMode.init.(*testingInitializer); ok {
		ti.Reset()
	}
	if ti, ok := SystemNamespace.init.(*testingInitializer); ok {
		ti.Reset()
	}
	if ti, ok := OperatorName.init.(*testingInitializer); ok {
		ti.Reset()
	}
}

func TestValueInitializer_GetWithContextTimeout(t *testing.T) {
	asserts := assert.New(t)

	resetAll()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	val, err := SystemNamespace.GetWithContext(ctx)
	asserts.Error(err, "expected error on timeout, got nil (val=%q)", val)
}

func TestValueInitializer_SetThenGet(t *testing.T) {
	asserts := assert.New(t)

	resetAll()
	SystemNamespace.Set("cattle-system")
	asserts.Equal("cattle-system", SystemNamespace.Get())
}

func TestValueInitializer_SetOnlyOnce(t *testing.T) {
	asserts := assert.New(t)

	resetAll()
	OperatorName.Set("first")
	OperatorName.Set("second")

	asserts.Equal("first", OperatorName.Get())
}

func TestValueInitializer_ConcurrentGetUnblocksAfterSet(t *testing.T) {
	asserts := assert.New(t)

	resetAll()
	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	results := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			results <- SystemNamespace.Get()
		}()
	}
	// Give goroutines time to start and block
	time.Sleep(20 * time.Millisecond)
	SystemNamespace.Set("ns")

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		// consume results and verify
		close(results)
		for r := range results {
			asserts.Equal("ns", r)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for Get goroutines to unblock after Set")
	}
}

func TestValueInitializer_DevModeBool(t *testing.T) {
	asserts := assert.New(t)

	resetAll()
	DevMode.Set(true)
	if !DevMode.Get() {
		t.Fatalf("DevMode.Get()=false; want true")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	val, err := DevMode.GetWithContext(ctx)
	asserts.NoError(err)
	asserts.True(val)
}
