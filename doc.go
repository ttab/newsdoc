package newsdoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// Document is a NewsDoc document.
type Document struct {
	// UUID is a unique ID for the document, this can for example be a
	// random v4 UUID, or a URI-derived v5 UUID.
	UUID string `json:"uuid,omitempty" jsonschema_extras:"format=uuid" proto:"1"`
	// Type is the content type of the document.
	Type string `json:"type,omitempty"  proto:"2"`
	// URI identifies the document (in a more human-readable way than the
	// UUID).
	URI string `json:"uri,omitempty" jsonschema_extras:"format=uri" proto:"3"`
	// URL is the browseable location of the document (if any).
	URL string `json:"url,omitempty" jsonschema_extras:"format=uri" proto:"4"`
	// Title is the title of the document, can be used as the document name,
	// or the headline when the document is displayed.
	Title string `json:"title,omitempty" proto:"5"`
	// Content is the content of the document, this is essentially what gets
	// rendered on the page when you view a document.
	Content []Block `json:"content,omitempty" proto:"6"`
	// Meta is the metadata for a document, this could be things like
	// teasers, open graph data, newsvalues.
	Meta []Block `json:"meta,omitempty" proto:"7"`
	// Links are links to other resources and entities. This could be links
	// to topics, categories and subjects for the document, or credited
	// authors.
	Links []Block `json:"links,omitempty" proto:"8"`
	// Language is the language used in the document as an IETF language
	// tag. F.ex. "en", "en-UK", "es", or "sv-SE".
	Language string `json:"language,omitempty" proto:"9"`
}

// Block is the building block for data embedded in documents. It is used for
// both content, links and metadata. Blocks have can be nested, but that's
// nothing to strive for, keep it simple.
type Block struct {
	// ID is the block ID,
	ID string `json:"id,omitempty" proto:"1"`
	// UUID is used to reference another Document in a block.
	UUID string `json:"uuid,omitempty" jsonschema_extras:"format=uuid" proto:"2"`
	// URI is used to reference another entity in a document.
	URI string `json:"uri,omitempty"  jsonschema_extras:"format=uri" proto:"3"`
	// URL is a browseable URL for the the block.
	URL string `json:"url,omitempty" jsonschema_extras:"format=uri" proto:"4"`
	// Type is the type of the block
	Type string `json:"type,omitempty" proto:"5"`
	// Title is the title/headline of the block, typically used in the
	// presentation of the block.
	Title string `json:"title,omitempty" proto:"6"`
	// Data contains block data.
	Data DataMap `json:"data,omitempty" proto:"7"`
	// Rel describes the relationship to the document/parent entity.
	Rel string `json:"rel,omitempty" proto:"8"`
	// Role is used either as an alternative to rel, or for nuancing the
	// relationship.
	Role string `json:"role,omitempty" proto:"9"`
	// Name is a name for the block. An alternative to "rel" when
	// relationship is a term that doesn't fit.
	Name string `json:"name,omitempty" proto:"10"`
	// Value is a value for the block. Useful when we want to store a
	// primitive value.
	Value string `json:"value,omitempty" proto:"11"`
	// ContentType is used to describe the content type of the block/linked
	// entity if it differs from the type of the block.
	Contenttype string `json:"contenttype,omitempty" proto:"12"`
	// Links are used to link to other resources and documents.
	Links []Block `json:"links,omitempty" proto:"13"`
	// Content is used to embed content blocks.
	Content []Block `json:"content,omitempty" proto:"14"`
	// Meta is used to embed metadata
	Meta []Block `json:"meta,omitempty" proto:"15"`
}

// DataMap is used as key -> (string) value data for blocks.
type DataMap map[string]string

// MarshalJSON implements a custom marshaler to make the JSON output of a
// document deterministic. Maps are unordered.
func (bd DataMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	keys := make([]string, 0, len(bd))

	for k := range bd {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	buf.WriteString("{")

	for i, k := range keys {
		if i != 0 {
			buf.WriteString(",")
		}

		// marshal key
		key, err := json.Marshal(k)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal data key: %w", err)
		}

		buf.Write(key)
		buf.WriteString(":")

		val, err := json.Marshal(bd[k])
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal data value: %w", err)
		}

		buf.Write(val)
	}

	buf.WriteString("}")

	return buf.Bytes(), nil
}
