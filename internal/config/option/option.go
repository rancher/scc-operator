package option

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/rancher/scc-operator/pkg/util/log"
)

var logger = log.NewComponentLogger("int/config/option")

type RegisteredOption interface {
	GetName() string
	GetDefaultAsString() string

	SetEnvKey(string)
	GetEnvKey() string
	GetEnv() string
	SetAllowFromEnv(bool)
	AllowsEnv() bool

	SetFlagKey(string)
	GetFlagKey() string
	SetAllowFromFlag(bool)
	AllowsFlag() bool

	SetConfigMapKey(string)
	GetConfigMapKey() string
	SetAllowFromConfigMap(bool)
	AllowsConfigMap() bool

	Type() reflect.Type
}

// Option is a reimagination of `Setting` from r/r codebase.
// This implementation is focused on centralizing various configuration sources of the "same value" (rather than holding the values).
type Option[T any] struct {
	// Name gives the option an identifier and is the only required field
	Name string

	// The NewOption constructor will initialize any unset keys based on the name via `prepareUnsetKeys`
	// You can manually configure these keys if the Name generated value is undesirable
	EnvKey       string
	FlagKey      string
	ConfigMapKey string

	// Default represents the default value when unset from all other sources.
	Default   T
	FlagValue T

	AllowFromEnv       bool
	AllowFromFlag      bool
	AllowFromConfigMap bool

	// TODO(improve/remove): figure out if we want these...it seems cool
	Parse    func(string) (T, error)
	Validate func(T) error
}

func (s *Option[T]) GetName() string {
	return s.Name
}

func (s *Option[T]) GetDefaultAsString() string {
	switch v := any(s.Default).(type) {
	case int:
		return strconv.Itoa(v)
	case float64:
		// Convert float with a specific precision
		return strconv.FormatFloat(v, 'f', 2, 64)
	case string:
		// If it's already a string, just return it
		return v
	default:
		// Fallback for all other types (structs, pointers, interfaces, etc.)
		return fmt.Sprintf("%v", s.Default)
	}
}

func (s *Option[T]) SetEnvKey(in string) {
	s.EnvKey = in
}

func (s *Option[T]) GetEnvKey() string {
	return s.EnvKey
}

func (s *Option[T]) GetEnv() string {
	if !s.AllowFromEnv {
		return ""
	}

	return os.Getenv(s.EnvKey)
}

func (s *Option[T]) SetAllowFromEnv(isAllowed bool) {
	s.AllowFromEnv = isAllowed
}

func (s *Option[T]) AllowsEnv() bool {
	return s.AllowFromEnv
}

// TODO(optional): could add a GetEnvValidated here...but this depends on Prase/Validate

func (s *Option[T]) SetFlagKey(in string) {
	s.FlagKey = in
}

func (s *Option[T]) GetFlagKey() string {
	return s.FlagKey
}

func (s *Option[T]) SetAllowFromFlag(isAllowed bool) {
	s.AllowFromFlag = isAllowed
}

func (s *Option[T]) AllowsFlag() bool {
	return s.AllowFromFlag
}

func (s *Option[T]) SetConfigMapKey(in string) {
	s.ConfigMapKey = in
}

func (s *Option[T]) GetConfigMapKey() string {
	return s.ConfigMapKey
}

func (s *Option[T]) SetAllowFromConfigMap(isAllowed bool) {
	s.AllowFromConfigMap = isAllowed
}

func (s *Option[T]) AllowsConfigMap() bool {
	return s.AllowFromConfigMap
}

func (s *Option[T]) Type() reflect.Type {
	var z T
	return reflect.TypeOf(z)
}

var _ RegisteredOption = &Option[string]{}

var (
	options = map[string]RegisteredOption{}
)

type OptionalValue func(RegisteredOption)

func WithEnvKey(key string) OptionalValue {
	return func(s RegisteredOption) {
		s.SetEnvKey(key)
	}
}

func WithFlagKey(key string) OptionalValue {
	return func(s RegisteredOption) {
		s.SetFlagKey(key)
	}
}

func WithConfigMapKey(key string) OptionalValue {
	return func(s RegisteredOption) {
		s.SetConfigMapKey(key)
		s.SetAllowFromConfigMap(true)
	}
}

var AllowedFromConfigMap OptionalValue = func(s RegisteredOption) {
	s.SetAllowFromConfigMap(true)
}

var WithoutEnv OptionalValue = func(s RegisteredOption) {
	s.SetEnvKey("")
	s.SetAllowFromEnv(false)
}

var WithoutFlag OptionalValue = func(s RegisteredOption) {
	s.SetFlagKey("")
	s.SetAllowFromFlag(false)
}

// NewOption will create and store a new operation.
// The `name` input should be a a kebab case string
func NewOption[T any](name string, defaultValue T, opts ...OptionalValue) *Option[T] {
	o := &Option[T]{
		Name:               name,
		Default:            defaultValue,
		AllowFromEnv:       true,
		AllowFromFlag:      true,
		AllowFromConfigMap: false,
	}
	for _, opt := range opts {
		opt(o)
	}

	prepareUnsetKeys(o)
	logger.Debugf("NewOption[%s] %v", name, o)

	options[o.GetName()] = o
	return o
}

func prepareUnsetKeys[T any](o *Option[T]) {
	if o.AllowFromEnv && o.EnvKey == "" {
		o.EnvKey = strings.ToUpper(strings.ReplaceAll(o.GetName(), "-", "_"))
	}
	if o.AllowFromFlag && o.FlagKey == "" {
		o.FlagKey = o.GetName()
	}
	if o.AllowFromConfigMap && o.ConfigMapKey == "" {
		o.ConfigMapKey = o.GetName()
	}
}
