package util

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

var (
	DevMode         = valueInitializer[bool]{}
	SystemNamespace = valueInitializer[string]{}
	OperatorName    = valueInitializer[string]{}
)


func init() {
	DevMode.Set(true)
}
