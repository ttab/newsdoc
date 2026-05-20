package newsdoc_test

import (
	"testing"

	"github.com/ttab/newsdoc"
)

func TestBlockMatchFunc(t *testing.T) {
	matcher := newsdoc.BlockMatchFunc(func(block newsdoc.Block) bool {
		return block.Value == "target"
	})

	if !matcher.Match(newsdoc.Block{Value: "target"}) {
		t.Error("should match block with Value 'target'")
	}

	if matcher.Match(newsdoc.Block{Value: "other"}) {
		t.Error("should not match block with Value 'other'")
	}
}

func TestBlockRole(t *testing.T) {
	role := newsdoc.BlockRole("heading")

	if !role.Match(newsdoc.Block{Role: "heading"}) {
		t.Error("should match block with Role 'heading'")
	}

	if role.Match(newsdoc.Block{Role: "body"}) {
		t.Error("should not match block with Role 'body'")
	}

	if role.Match(newsdoc.Block{}) {
		t.Error("should not match block with empty Role")
	}
}

func TestBlocksWithType(t *testing.T) {
	matcher := newsdoc.BlocksWithType("core/text")

	if !matcher.Match(newsdoc.Block{Type: "core/text"}) {
		t.Error("should match block with matching type")
	}

	if matcher.Match(newsdoc.Block{Type: "core/image"}) {
		t.Error("should not match block with different type")
	}

	if matcher.Match(newsdoc.Block{}) {
		t.Error("should not match block with empty type")
	}
}

func TestBlocksWithRel(t *testing.T) {
	matcher := newsdoc.BlocksWithRel("author")

	if !matcher.Match(newsdoc.Block{Rel: "author"}) {
		t.Error("should match block with matching rel")
	}

	if matcher.Match(newsdoc.Block{Rel: "subject"}) {
		t.Error("should not match block with different rel")
	}

	// Should match regardless of type.
	if !matcher.Match(newsdoc.Block{Type: "core/person", Rel: "author"}) {
		t.Error("should match block with matching rel regardless of type")
	}
}

func TestBlocksWithTypeAndRel(t *testing.T) {
	matcher := newsdoc.BlocksWithTypeAndRel("core/person", "author")

	if !matcher.Match(newsdoc.Block{Type: "core/person", Rel: "author"}) {
		t.Error("should match block with matching type and rel")
	}

	if matcher.Match(newsdoc.Block{Type: "core/person", Rel: "subject"}) {
		t.Error("should not match when rel differs")
	}

	if matcher.Match(newsdoc.Block{Type: "core/org", Rel: "author"}) {
		t.Error("should not match when type differs")
	}

	if matcher.Match(newsdoc.Block{Type: "core/person"}) {
		t.Error("should not match when rel is empty")
	}
}

func TestBlocksWithTypeAndRole(t *testing.T) {
	matcher := newsdoc.BlocksWithTypeAndRole("core/text", "heading")

	if !matcher.Match(newsdoc.Block{Type: "core/text", Role: "heading"}) {
		t.Error("should match block with matching type and role")
	}

	if matcher.Match(newsdoc.Block{Type: "core/text", Role: "body"}) {
		t.Error("should not match when role differs")
	}

	if matcher.Match(newsdoc.Block{Type: "core/image", Role: "heading"}) {
		t.Error("should not match when type differs")
	}
}

func TestBlockMatchesAll(t *testing.T) {
	matcher := newsdoc.BlockMatchesAll(
		newsdoc.BlocksWithType("core/text"),
		newsdoc.BlockRole("heading"),
	)

	if !matcher.Match(newsdoc.Block{Type: "core/text", Role: "heading"}) {
		t.Error("should match when all conditions met")
	}

	if matcher.Match(newsdoc.Block{Type: "core/text", Role: "body"}) {
		t.Error("should not match when only some conditions met")
	}

	if matcher.Match(newsdoc.Block{Type: "core/image", Role: "heading"}) {
		t.Error("should not match when only some conditions met")
	}
}

func TestBlockMatchesAllEmpty(t *testing.T) {
	matcher := newsdoc.BlockMatchesAll()

	if !matcher.Match(newsdoc.Block{Type: "anything"}) {
		t.Error("empty BlockMatchesAll should match everything")
	}
}

func TestBlockMatchesAny(t *testing.T) {
	matcher := newsdoc.BlockMatchesAny(
		newsdoc.BlocksWithType("core/text"),
		newsdoc.BlocksWithType("core/image"),
	)

	if !matcher.Match(newsdoc.Block{Type: "core/text"}) {
		t.Error("should match first condition")
	}

	if !matcher.Match(newsdoc.Block{Type: "core/image"}) {
		t.Error("should match second condition")
	}

	if matcher.Match(newsdoc.Block{Type: "core/video"}) {
		t.Error("should not match when no conditions met")
	}
}

func TestBlockMatchesAnyEmpty(t *testing.T) {
	matcher := newsdoc.BlockMatchesAny()

	if matcher.Match(newsdoc.Block{Type: "anything"}) {
		t.Error("empty BlockMatchesAny should match nothing")
	}
}

func TestBlockDoesntMatch(t *testing.T) {
	matcher := newsdoc.BlockDoesntMatch(newsdoc.BlocksWithType("core/text"))

	if matcher.Match(newsdoc.Block{Type: "core/text"}) {
		t.Error("should not match negated condition")
	}

	if !matcher.Match(newsdoc.Block{Type: "core/image"}) {
		t.Error("should match when condition is not met")
	}
}

func TestBlockMatcherComposition(t *testing.T) {
	// Match (type=core/text AND role=heading) OR type=core/image.
	matcher := newsdoc.BlockMatchesAny(
		newsdoc.BlockMatchesAll(
			newsdoc.BlocksWithType("core/text"),
			newsdoc.BlockRole("heading"),
		),
		newsdoc.BlocksWithType("core/image"),
	)

	if !matcher.Match(newsdoc.Block{Type: "core/text", Role: "heading"}) {
		t.Error("should match first branch")
	}

	if !matcher.Match(newsdoc.Block{Type: "core/image"}) {
		t.Error("should match second branch")
	}

	if matcher.Match(newsdoc.Block{Type: "core/text", Role: "body"}) {
		t.Error("should not match text without heading role")
	}
}
