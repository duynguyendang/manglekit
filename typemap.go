package manglekit

import (
	"fmt"
	"reflect"
)

// Bidirectional maps: provider name <-> options pointer type (*T)
var (
	nameToOptionsType = make(map[string]reflect.Type)
	optionsTypeToName = make(map[reflect.Type]string)
)

// RegisterOptions registers the **pointer-to-struct** options type for a provider.
// Always pass a **typed nil pointer**:  (*MyOptions)(nil)
func RegisterOptions(providerName string, typedNilPtr any) error {
	t := reflect.TypeOf(typedNilPtr)
	if t == nil {
		return fmt.Errorf("RegisterOptions %q: got nil; pass a typed nil pointer like (*T)(nil)", providerName)
	}
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("RegisterOptions %q: expected pointer to struct, got %v", providerName, t)
	}
	nameToOptionsType[providerName] = t
	optionsTypeToName[t] = providerName
	return nil
}
