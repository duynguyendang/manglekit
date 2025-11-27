// Copyright 2024 MangleKube.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fn

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// ActionFunc is a generic adapter that wraps a Go function to implement the core.Action interface.
type ActionFunc[I any, O any] struct {
	fn   func(context.Context, I) (O, error)
	name string
}

// New creates a new ActionFunc.
func New[I any, O any](name string, fn func(context.Context, I) (O, error)) *ActionFunc[I, O] {
	return &ActionFunc[I, O]{
		fn:   fn,
		name: name,
	}
}

// Execute implements the core.Action interface.
func (a *ActionFunc[I, O]) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	payload, ok := input.Payload.(I)
	if !ok {
		return core.Envelope{}, fmt.Errorf("unexpected input type: %T", input.Payload)
	}
	output, err := a.fn(ctx, payload)
	if err != nil {
		return core.Envelope{}, err
	}
	return core.NewEnvelope(output), nil
}

// Metadata implements the core.Action interface.
func (a *ActionFunc[I, O]) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: a.name,
		Type: "function",
	}
}
