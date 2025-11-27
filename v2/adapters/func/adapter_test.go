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

package fn_test

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	fn "github.com/duynguyendang/manglekit/v2/adapters/func"
	"github.com/stretchr/testify/assert"
)

func TestActionFunc(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful execution", func(t *testing.T) {
		// Define a simple function to wrap
		addOne := func(ctx context.Context, i int) (int, error) {
			return i + 1, nil
		}

		// Create the adapter
		action := fn.New("addOne", addOne)

		// Create input envelope
		input := core.NewEnvelope(10)

		// Execute the action
		output, err := action.Execute(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, output)

		// Check the output payload
		result, ok := output.Payload.(int)
		assert.True(t, ok)
		assert.Equal(t, 11, result)

		// Check metadata
		meta := action.Metadata()
		assert.Equal(t, "addOne", meta.Name)
		assert.Equal(t, "function", meta.Type)
	})

	t.Run("Function returns an error", func(t *testing.T) {
		// Define a function that always returns an error
		failFunc := func(ctx context.Context, s string) (string, error) {
			return "", errors.New("something went wrong")
		}

		action := fn.New("failFunc", failFunc)
		input := core.NewEnvelope("test")

		output, err := action.Execute(ctx, input)
		assert.Error(t, err)
		assert.EqualError(t, err, "something went wrong")
		assert.Equal(t, core.Envelope{}, output) // Expect a zero-value envelope on error
	})

	t.Run("Input type mismatch", func(t *testing.T) {
		addOne := func(ctx context.Context, i int) (int, error) {
			return i + 1, nil
		}

		action := fn.New("addOne", addOne)

		// Pass a string instead of an int
		input := core.NewEnvelope("not-an-integer")

		output, err := action.Execute(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected input type")
		assert.Equal(t, core.Envelope{}, output) // Expect a zero-value envelope on error
	})
}
