package newsdoc_test

import (
	"encoding/json"
	"testing"

	"github.com/ttab/newsdoc"
)

func TestJSONSchema(t *testing.T) {
	schema := newsdoc.JSONSchema()

	if len(schema) == 0 {
		t.Fatal("JSONSchema returned empty bytes")
	}

	// Verify it's valid JSON.
	var parsed map[string]interface{}

	err := json.Unmarshal(schema, &parsed)
	if err != nil {
		t.Fatalf("JSONSchema is not valid JSON: %v", err)
	}

	// Check that it looks like a JSON Schema.
	if _, ok := parsed["$schema"]; !ok {
		t.Error("expected '$schema' key in JSON schema")
	}
}
