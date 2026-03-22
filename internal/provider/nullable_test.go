package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type testNullable[T any] struct {
	isSet bool
	value *T
}

func (n testNullable[T]) IsSet() bool { return n.isSet }
func (n testNullable[T]) Get() *T     { return n.value }

func TestNullableStringToTF(t *testing.T) {
	value := "hello"

	tests := []struct {
		name     string
		input    testNullable[string]
		expected types.String
	}{
		{name: "unset", input: testNullable[string]{isSet: false}, expected: types.StringNull()},
		{name: "explicit null", input: testNullable[string]{isSet: true, value: nil}, expected: types.StringNull()},
		{name: "value", input: testNullable[string]{isSet: true, value: &value}, expected: types.StringValue("hello")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullableStringToTF(tt.input)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestRequiredNullableString(t *testing.T) {
	value := "id-123"

	if got, ok := requiredNullableString(testNullable[string]{isSet: false}); ok || got != "" {
		t.Fatalf("expected unset to fail, got (%q, %v)", got, ok)
	}
	if got, ok := requiredNullableString(testNullable[string]{isSet: true, value: nil}); ok || got != "" {
		t.Fatalf("expected explicit null to fail, got (%q, %v)", got, ok)
	}
	if got, ok := requiredNullableString(testNullable[string]{isSet: true, value: &value}); !ok || got != value {
		t.Fatalf("expected (%q, true), got (%q, %v)", value, got, ok)
	}
}
