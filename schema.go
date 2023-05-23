package newsdoc

import _ "embed"

//go:embed newsdoc.schema.json
var schema string

// JSONSchema returns the NewsDoc JSON schema.
func JSONSchema() []byte {
	return []byte(schema)
}
