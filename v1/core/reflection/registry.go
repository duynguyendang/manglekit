package reflection

import (
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/google/mangle/ast"
)

// ConverterFunc converts a specific reflect.Value into a Mangle Constant.
type ConverterFunc func(v reflect.Value) (ast.Constant, error)

var (
	// Default hooks for standard library types
	defaultHooks = map[reflect.Type]ConverterFunc{
		reflect.TypeOf(time.Time{}): timeConverter,
		reflect.TypeOf(net.IP{}):    ipConverter,
	}
	// Allow users to register custom hooks
	customHooks = make(map[reflect.Type]ConverterFunc)
	mu          sync.RWMutex
)

// RegisterHook allows for registering a custom type converter.
// It is thread-safe, but recommended to be called during initialization.
func RegisterHook(t reflect.Type, fn ConverterFunc) {
	mu.Lock()
	defer mu.Unlock()
	customHooks[t] = fn
}

func getHook(t reflect.Type) ConverterFunc {
	mu.RLock()
	defer mu.RUnlock()
	if fn, ok := customHooks[t]; ok {
		return fn
	}
	if fn, ok := defaultHooks[t]; ok {
		return fn
	}
	return nil
}

func timeConverter(v reflect.Value) (ast.Constant, error) {
	t := v.Interface().(time.Time)
	return ast.Number(t.Unix()), nil
}

func ipConverter(v reflect.Value) (ast.Constant, error) {
	ip := v.Interface().(net.IP)
	return ast.String(ip.String()), nil
}
