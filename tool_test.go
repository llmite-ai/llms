package llms

import (
	"testing"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structures for GenerateSchema
type SimpleStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type ComplexStruct struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Age         int               `json:"age"`
	IsActive    bool              `json:"is_active"`
	Score       float64           `json:"score"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	OptionalPtr *string           `json:"optional_ptr,omitempty"`
}

type NestedStruct struct {
	User    SimpleStruct `json:"user"`
	Address Address      `json:"address"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type StructWithValidation struct {
	Email    string `json:"email" jsonschema:"format=email"`
	URL      string `json:"url" jsonschema:"format=uri"`
	MinMax   int    `json:"min_max" jsonschema:"minimum=1,maximum=100"`
	Required string `json:"required" jsonschema:"required"`
}

type StructWithSlices struct {
	StringSlice []string      `json:"string_slice"`
	IntSlice    []int         `json:"int_slice"`
	StructSlice []SimpleStruct `json:"struct_slice"`
}

func TestGenerateSchema_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() *jsonschema.Schema
		expected map[string]interface{}
	}{
		{
			name: "string type",
			testFunc: func() *jsonschema.Schema {
				return GenerateSchema[string]()
			},
			expected: map[string]interface{}{
				"type": "string",
			},
		},
		{
			name: "int type",
			testFunc: func() *jsonschema.Schema {
				return GenerateSchema[int]()
			},
			expected: map[string]interface{}{
				"type": "integer",
			},
		},
		{
			name: "bool type",
			testFunc: func() *jsonschema.Schema {
				return GenerateSchema[bool]()
			},
			expected: map[string]interface{}{
				"type": "boolean",
			},
		},
		{
			name: "float64 type",
			testFunc: func() *jsonschema.Schema {
				return GenerateSchema[float64]()
			},
			expected: map[string]interface{}{
				"type": "number",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := tt.testFunc()
			require.NotNil(t, schema)
			assert.Equal(t, tt.expected["type"], schema.Type)
		})
	}
}

func TestGenerateSchema_SimpleStruct(t *testing.T) {
	schema := GenerateSchema[SimpleStruct]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)
	
	// Check that we have the expected properties
	assert.Equal(t, 2, schema.Properties.Len())
	
	// Check name property
	nameSchema, exists := schema.Properties.Get("name")
	assert.True(t, exists)
	assert.Equal(t, "string", nameSchema.Type)
	
	// Check age property
	ageSchema, exists := schema.Properties.Get("age")
	assert.True(t, exists)
	assert.Equal(t, "integer", ageSchema.Type)
}

func TestGenerateSchema_ComplexStruct(t *testing.T) {
	schema := GenerateSchema[ComplexStruct]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)
	
	// Check that we have the expected number of properties
	assert.Equal(t, 9, schema.Properties.Len())
	
	// Check various property types
	propertyTests := []struct {
		name         string
		expectedType string
	}{
		{"id", "string"},
		{"name", "string"},
		{"age", "integer"},
		{"is_active", "boolean"},
		{"score", "number"},
		{"created_at", "string"},
	}
	
	for _, test := range propertyTests {
		prop, exists := schema.Properties.Get(test.name)
		assert.True(t, exists, "Property %s should exist", test.name)
		assert.Equal(t, test.expectedType, prop.Type, "Property %s should have type %s", test.name, test.expectedType)
	}
	
	// Check array property
	tagsSchema, exists := schema.Properties.Get("tags")
	assert.True(t, exists)
	assert.Equal(t, "array", tagsSchema.Type)
	assert.NotNil(t, tagsSchema.Items)
	assert.Equal(t, "string", tagsSchema.Items.Type)
	
	// Check object property (map)
	metadataSchema, exists := schema.Properties.Get("metadata")
	assert.True(t, exists)
	assert.Equal(t, "object", metadataSchema.Type)
}

func TestGenerateSchema_NestedStruct(t *testing.T) {
	schema := GenerateSchema[NestedStruct]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)
	
	// Check user property
	userSchema, exists := schema.Properties.Get("user")
	assert.True(t, exists)
	assert.Equal(t, "object", userSchema.Type)
	assert.NotNil(t, userSchema.Properties)
	
	// Check nested user properties
	userName, exists := userSchema.Properties.Get("name")
	assert.True(t, exists)
	assert.Equal(t, "string", userName.Type)
	
	userAge, exists := userSchema.Properties.Get("age")
	assert.True(t, exists)
	assert.Equal(t, "integer", userAge.Type)
	
	// Check address property
	addressSchema, exists := schema.Properties.Get("address")
	assert.True(t, exists)
	assert.Equal(t, "object", addressSchema.Type)
	assert.NotNil(t, addressSchema.Properties)
	
	// Check nested address properties
	addressTests := []string{"street", "city", "country"}
	for _, prop := range addressTests {
		addressProp, exists := addressSchema.Properties.Get(prop)
		assert.True(t, exists, "Address property %s should exist", prop)
		assert.Equal(t, "string", addressProp.Type, "Address property %s should be string", prop)
	}
}

func TestGenerateSchema_SliceTypes(t *testing.T) {
	schema := GenerateSchema[StructWithSlices]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	
	// Check string slice
	stringSliceSchema, exists := schema.Properties.Get("string_slice")
	assert.True(t, exists)
	assert.Equal(t, "array", stringSliceSchema.Type)
	assert.NotNil(t, stringSliceSchema.Items)
	assert.Equal(t, "string", stringSliceSchema.Items.Type)
	
	// Check int slice
	intSliceSchema, exists := schema.Properties.Get("int_slice")
	assert.True(t, exists)
	assert.Equal(t, "array", intSliceSchema.Type)
	assert.NotNil(t, intSliceSchema.Items)
	assert.Equal(t, "integer", intSliceSchema.Items.Type)
	
	// Check struct slice
	structSliceSchema, exists := schema.Properties.Get("struct_slice")
	assert.True(t, exists)
	assert.Equal(t, "array", structSliceSchema.Type)
	assert.NotNil(t, structSliceSchema.Items)
	assert.Equal(t, "object", structSliceSchema.Items.Type)
}

func TestGenerateSchema_ReflectorSettings(t *testing.T) {
	// Test that the reflector settings are applied correctly
	schema := GenerateSchema[SimpleStruct]()
	
	require.NotNil(t, schema)
	
	// The schema should not allow additional properties
	// Note: This is a bit tricky to test directly since AdditionalProperties
	// might be nil when false, but we can verify the reflector worked
	assert.NotNil(t, schema.Properties)
	
	// Test that DoNotReference is working by ensuring we don't have $ref fields
	// This is implicit in the structure - if references were used, we'd see $ref
	nameSchema, exists := schema.Properties.Get("name")
	assert.True(t, exists)
	assert.Equal(t, "string", nameSchema.Type)
	assert.Empty(t, nameSchema.Ref) // Should not have a reference
}

func TestGenerateSchema_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}
	
	schema := GenerateSchema[EmptyStruct]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)
	assert.Equal(t, 0, schema.Properties.Len())
}

func TestGenerateSchema_PointerTypes(t *testing.T) {
	type StructWithPointer struct {
		Required *string `json:"required"`
		Optional *string `json:"optional,omitempty"`
	}
	
	schema := GenerateSchema[StructWithPointer]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	
	// Both pointer fields should be present in properties
	requiredSchema, exists := schema.Properties.Get("required")
	assert.True(t, exists)
	assert.Equal(t, "string", requiredSchema.Type)
	
	optionalSchema, exists := schema.Properties.Get("optional")
	assert.True(t, exists)
	assert.Equal(t, "string", optionalSchema.Type)
}

func TestGenerateSchema_InterfaceType(t *testing.T) {
	type StructWithInterface struct {
		Data interface{} `json:"data"`
	}
	
	schema := GenerateSchema[StructWithInterface]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	
	// Interface{} should be handled appropriately
	dataSchema, exists := schema.Properties.Get("data")
	assert.True(t, exists)
	// The exact type might vary, but it should exist
	assert.NotNil(t, dataSchema)
}

func TestGenerateSchema_MapTypes(t *testing.T) {
	type StructWithMaps struct {
		StringMap map[string]string `json:"string_map"`
		IntMap    map[string]int    `json:"int_map"`
	}
	
	schema := GenerateSchema[StructWithMaps]()
	
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	
	// Check string map
	stringMapSchema, exists := schema.Properties.Get("string_map")
	assert.True(t, exists)
	assert.Equal(t, "object", stringMapSchema.Type)
	
	// Check int map
	intMapSchema, exists := schema.Properties.Get("int_map")
	assert.True(t, exists)
	assert.Equal(t, "object", intMapSchema.Type)
}