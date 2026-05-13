package newsdoc_test

import (
	"testing"

	"github.com/ttab/newsdoc"
)

const modified = "modified"

func TestDocumentClone(t *testing.T) {
	original := newsdoc.Document{
		UUID:     "doc-uuid",
		Type:     "core/article",
		URI:      "article://test",
		URL:      "https://example.com/article",
		Title:    "Test Article",
		Language: "en",
		Meta: []newsdoc.Block{
			{Type: "core/newsvalue", Data: newsdoc.DataMap{"score": "5"}},
		},
		Links: []newsdoc.Block{
			{Type: "core/author", Rel: "author", Title: "Jane"},
		},
		Content: []newsdoc.Block{
			{Type: "core/text", Role: "heading", Value: "Hello"},
		},
	}

	clone := original.Clone()

	// Scalar fields should be equal.
	if clone.UUID != original.UUID {
		t.Errorf("UUID mismatch: got %q, want %q", clone.UUID, original.UUID)
	}

	if clone.Title != original.Title {
		t.Errorf("Title mismatch: got %q, want %q", clone.Title, original.Title)
	}

	// Slices should be separate copies.
	clone.Meta[0].Type = modified

	if original.Meta[0].Type == modified {
		t.Error("modifying clone.Meta affected the original")
	}

	clone.Links[0].Title = modified

	if original.Links[0].Title == modified {
		t.Error("modifying clone.Links affected the original")
	}

	clone.Content[0].Value = modified

	if original.Content[0].Value == modified {
		t.Error("modifying clone.Content affected the original")
	}

	// Data maps should be separate copies.
	clone.Meta = original.Clone().Meta
	clone.Meta[0].Data["score"] = "99"

	if original.Meta[0].Data["score"] == "99" {
		t.Error("modifying cloned Data map affected the original")
	}
}

func TestBlockClone(t *testing.T) {
	original := newsdoc.Block{
		ID:          "block-1",
		UUID:        "block-uuid",
		URI:         "block://test",
		URL:         "https://example.com/block",
		Type:        "core/text",
		Title:       "Block Title",
		Rel:         "item",
		Role:        "heading",
		Name:        "intro",
		Value:       "Hello",
		Contenttype: "text/plain",
		Sensitivity: "internal",
		Data:        newsdoc.DataMap{"key": "value"},
		Meta: []newsdoc.Block{
			{Type: "core/meta", Data: newsdoc.DataMap{"m": "1"}},
		},
		Links: []newsdoc.Block{
			{Type: "core/link", Rel: "related"},
		},
		Content: []newsdoc.Block{
			{Type: "core/text", Value: "nested"},
		},
	}

	clone := original.Clone()

	// Scalar fields should match.
	if clone.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", clone.ID, original.ID)
	}

	if clone.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", clone.Type, original.Type)
	}

	// Data should be a deep copy.
	clone.Data["key"] = modified

	if original.Data["key"] == modified {
		t.Error("modifying cloned block Data affected the original")
	}

	// Nested block slices should be deep copies.
	clone.Meta[0].Data["m"] = modified

	if original.Meta[0].Data["m"] == modified {
		t.Error("modifying cloned block Meta affected the original")
	}

	clone.Links[0].Rel = modified

	if original.Links[0].Rel == modified {
		t.Error("modifying cloned block Links affected the original")
	}

	clone.Content[0].Value = modified

	if original.Content[0].Value == modified {
		t.Error("modifying cloned block Content affected the original")
	}
}

func TestCloneEmptyDocument(t *testing.T) {
	original := newsdoc.Document{}
	clone := original.Clone()

	if clone.UUID != "" || clone.Type != "" || clone.Title != "" {
		t.Error("cloning empty document should produce empty document")
	}

	// cloneBlocks always allocates a new slice, even for nil input.
	if len(clone.Meta) != 0 || len(clone.Links) != 0 || len(clone.Content) != 0 {
		t.Error("cloning empty document should produce empty block slices")
	}
}

func TestCloneBlockNilData(t *testing.T) {
	original := newsdoc.Block{Type: "core/text"}
	clone := original.Clone()

	if clone.Data != nil {
		t.Error("cloning block with nil Data should produce nil Data")
	}
}
