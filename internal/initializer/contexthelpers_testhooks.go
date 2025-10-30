//go:build test

package initializer

// SetForTest forcibly sets the value regardless of prior initialization.
// This is intended for use in tests across packages that need to change
// the value multiple times in different test cases.
func (v *valueInitializer[T]) SetForTest(newValue T) {
	if ih, ok := v.init.(*InitHandler); ok {
		ih.checkInitCond()
		ih.initCond.L.Lock()
		v.value = newValue
		ih.initialized = true
		ih.initCond.Broadcast()
		ih.initCond.L.Unlock()
		return
	}
	// Fallback: if a different Initializer is used, best effort via Set
	v.Set(newValue)
}
