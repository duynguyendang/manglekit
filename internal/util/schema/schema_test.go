package schema_test

import (
	"testing"

	"github.com/duynguyendang/manglekit-wip/internal/util/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestGenerate(t *testing.T) {
	output, err := schema.Generate(User{})
	require.NoError(t, err)

	assert.Contains(t, output, "properties")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "age")
}

func TestValidateStruct(t *testing.T) {
	// User struct is defined at package level

	// First generate the schema
	schemaStr, err := schema.Generate(User{})
	require.NoError(t, err)

	t.Run("Valid", func(t *testing.T) {
		user := User{Name: "Alice", Age: 30}
		err := schema.ValidateStruct(schemaStr, user)
		assert.NoError(t, err)
	})

	// Note: ValidateStruct marshals the struct to JSON.
	// Since Go is statically typed, we can't easily pass a "wrong type" struct
	// that matches the shape but has wrong types unless we use a map or different struct.
	// But ValidateStruct takes 'any', so let's try passing a different struct that doesn't match the schema constraints
	// or using ValidateJSON for type mismatch tests as requested.
}

func TestValidateJSON(t *testing.T) {
	// User struct is defined at package level

	schemaStr, err := schema.Generate(User{})
	require.NoError(t, err)

	t.Run("Invalid Type", func(t *testing.T) {
		jsonStr := `{"name": "Alice", "age": "wrong_type"}`
		err := schema.ValidateJSON(schemaStr, jsonStr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Valid", func(t *testing.T) {
		jsonStr := `{"name": "Bob", "age": 25}`
		err := schema.ValidateJSON(schemaStr, jsonStr)
		assert.NoError(t, err)
	})
}
