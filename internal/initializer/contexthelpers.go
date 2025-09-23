package initializer

import "context"

type valueInitializer[T any] struct {
	value T
	init  Initializer
}

func (v *valueInitializer[T]) Set(newValue T) {
	v.init.InitOnce(func() {
		v.value = newValue
	})
}

func (v *valueInitializer[T]) Get() T {
	v.init.WaitForInit()
	return v.value
}

func (v *valueInitializer[T]) GetWithContext(ctx context.Context) (T, error) {
	err := v.init.WaitForInitContext(ctx)
	if err != nil {
		var zeroVal T
		return zeroVal, err
	}
	return v.value, err
}

var (
	DevMode      = valueInitializer[bool]{init: &InitHandler{}}
	OperatorName = valueInitializer[string]{init: &InitHandler{}}
)
