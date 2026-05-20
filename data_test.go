package newsdoc_test

import (
	"encoding/json"
	"testing"

	"github.com/ttab/newsdoc"
)

const (
	testValue = "value"
	testYes   = "yes"
)

func TestDataMapGet(t *testing.T) {
	dm := newsdoc.DataMap{"key": testValue, "empty": ""}

	if got := dm.Get("key", "default"); got != testValue {
		t.Errorf("Get existing key: got %q, want %q", got, testValue)
	}

	if got := dm.Get("empty", "default"); got != "" {
		t.Errorf("Get empty value: got %q, want %q", got, "")
	}

	if got := dm.Get("missing", "default"); got != "default" {
		t.Errorf("Get missing key: got %q, want %q", got, "default")
	}
}

func TestDataMapGetNil(t *testing.T) {
	var dm newsdoc.DataMap

	if got := dm.Get("key", "fallback"); got != "fallback" {
		t.Errorf("Get on nil DataMap: got %q, want %q", got, "fallback")
	}
}

func TestDataMapDelete(t *testing.T) {
	dm := newsdoc.DataMap{"a": "1", "b": "2", "c": "3"}

	dm.Delete("a", "c")

	if _, ok := dm["a"]; ok {
		t.Error("key 'a' should have been deleted")
	}

	if _, ok := dm["c"]; ok {
		t.Error("key 'c' should have been deleted")
	}

	if dm["b"] != "2" {
		t.Error("key 'b' should be unchanged")
	}
}

func TestDataMapDeleteNil(t *testing.T) {
	var dm newsdoc.DataMap

	// Should not panic on nil DataMap.
	dm.Delete("key")

	if dm != nil {
		t.Error("nil DataMap should remain nil after Delete")
	}
}

func TestDataMapDeleteNonexistent(t *testing.T) {
	dm := newsdoc.DataMap{"a": "1"}

	// Should not panic when deleting non-existent keys.
	dm.Delete("nonexistent")

	if dm["a"] != "1" {
		t.Error("existing key should be unchanged")
	}
}

func TestDataMapDropEmpty(t *testing.T) {
	dm := newsdoc.DataMap{"keep": testValue, "drop": "", "also_keep": testYes}

	dm.DropEmpty()

	if _, ok := dm["drop"]; ok {
		t.Error("empty value key should have been dropped")
	}

	if dm["keep"] != testValue {
		t.Error("non-empty key 'keep' should remain")
	}

	if dm["also_keep"] != testYes {
		t.Error("non-empty key 'also_keep' should remain")
	}
}

func TestDataMapDropEmptyNil(t *testing.T) {
	var dm newsdoc.DataMap

	// Should not panic on nil DataMap.
	dm.DropEmpty()

	if dm != nil {
		t.Error("nil DataMap should remain nil after DropEmpty")
	}
}

func TestDataMapDropEmptyAllEmpty(t *testing.T) {
	dm := newsdoc.DataMap{"a": "", "b": ""}

	dm.DropEmpty()

	if len(dm) != 0 {
		t.Errorf("all keys should be dropped, got %d remaining", len(dm))
	}
}

func TestUpsertData(t *testing.T) {
	data := newsdoc.DataMap{"a": "1"}
	result := newsdoc.UpsertData(data, newsdoc.DataMap{"b": "2", "a": "overwritten"})

	if result["a"] != "overwritten" {
		t.Errorf("expected 'a' to be overwritten, got %q", result["a"])
	}

	if result["b"] != "2" {
		t.Errorf("expected 'b' to be '2', got %q", result["b"])
	}
}

func TestUpsertDataNilTarget(t *testing.T) {
	result := newsdoc.UpsertData(nil, newsdoc.DataMap{"key": testValue})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result["key"] != testValue {
		t.Errorf("expected 'key' to be 'value', got %q", result["key"])
	}
}

func TestUpsertDataEmptyNew(t *testing.T) {
	data := newsdoc.DataMap{"a": "1"}
	result := newsdoc.UpsertData(data, newsdoc.DataMap{})

	if result["a"] != "1" {
		t.Errorf("existing data should be preserved, got %q", result["a"])
	}
}

func TestDataWithDefaults(t *testing.T) {
	data := newsdoc.DataMap{"existing": "keep", "empty": ""}
	defaults := newsdoc.DataMap{"existing": "ignored", "empty": "filled", "new": "added"}

	result := newsdoc.DataWithDefaults(data, defaults)

	if result["existing"] != "keep" {
		t.Errorf("existing non-empty key should be kept, got %q", result["existing"])
	}

	if result["empty"] != "filled" {
		t.Errorf("empty key should be filled with default, got %q", result["empty"])
	}

	if result["new"] != "added" {
		t.Errorf("missing key should be added from defaults, got %q", result["new"])
	}
}

func TestDataWithDefaultsNilTarget(t *testing.T) {
	result := newsdoc.DataWithDefaults(nil, newsdoc.DataMap{"key": testValue})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result["key"] != testValue {
		t.Errorf("expected 'key' to be 'value', got %q", result["key"])
	}
}

func TestCopyData(t *testing.T) {
	src := newsdoc.DataMap{"a": "1", "b": "2", "c": "3"}
	dst := newsdoc.DataMap{"existing": testYes}

	result := newsdoc.CopyData(dst, src, "a", "c", "missing")

	if result["a"] != "1" {
		t.Errorf("'a' should be copied, got %q", result["a"])
	}

	if result["c"] != "3" {
		t.Errorf("'c' should be copied, got %q", result["c"])
	}

	if _, ok := result["missing"]; ok {
		t.Error("'missing' should not be in result")
	}

	if _, ok := result["b"]; ok {
		t.Error("'b' should not be copied (not in keys list)")
	}

	if result["existing"] != testYes {
		t.Error("existing dst data should be preserved")
	}
}

func TestCopyDataNilDst(t *testing.T) {
	src := newsdoc.DataMap{"a": "1"}
	result := newsdoc.CopyData(nil, src, "a")

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result["a"] != "1" {
		t.Errorf("'a' should be copied, got %q", result["a"])
	}
}

func TestCopyDataNilSrc(t *testing.T) {
	dst := newsdoc.DataMap{"keep": testYes}
	result := newsdoc.CopyData(dst, nil, "keep")

	if result["keep"] != testYes {
		t.Error("dst should be returned unchanged")
	}
}

func TestCopyDataBothNil(t *testing.T) {
	result := newsdoc.CopyData(nil, nil, "a")

	if result == nil {
		t.Fatal("result should not be nil even with nil inputs")
	}
}

func TestDataMapMarshalJSON(t *testing.T) {
	dm := newsdoc.DataMap{"b": "2", "a": "1"}

	data, err := json.Marshal(dm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Keys should be sorted deterministically.
	want := `{"a":"1","b":"2"}`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestDataMapMarshalJSONNil(t *testing.T) {
	var dm newsdoc.DataMap

	data, err := json.Marshal(dm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != "null" {
		t.Errorf("nil DataMap should marshal to null, got %s", data)
	}
}

func TestDataMapMarshalJSONEmpty(t *testing.T) {
	dm := newsdoc.DataMap{}

	data, err := json.Marshal(dm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != "{}" {
		t.Errorf("empty DataMap should marshal to {}, got %s", data)
	}
}

func TestDataMapMarshalJSONSingleKey(t *testing.T) {
	dm := newsdoc.DataMap{"only": "one"}

	data, err := json.Marshal(dm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"only":"one"}`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestDataMapRoundTrip(t *testing.T) {
	original := newsdoc.DataMap{"z": "last", "a": "first", "m": "middle"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored newsdoc.DataMap

	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for k, v := range original {
		if restored[k] != v {
			t.Errorf("key %q: got %q, want %q", k, restored[k], v)
		}
	}
}
