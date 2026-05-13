package newsdoc_test

import (
	"testing"

	"github.com/ttab/newsdoc"
)

const (
	altered      = "altered"
	firstAltered = "first-altered"
	newVideo     = "New Video"
)

func sampleBlocks() []newsdoc.Block {
	return []newsdoc.Block{
		{Type: coreText, Role: "heading", Value: "Title"},
		{Type: coreText, Role: "body", Value: "Paragraph 1"},
		{Type: "core/image", Rel: "photo", Title: "Image"},
		{Type: coreText, Role: "body", Value: "Paragraph 2"},
		{Type: "core/embed", Value: "https://example.com"},
	}
}

func TestFirstBlock(t *testing.T) {
	blocks := sampleBlocks()

	b, ok := newsdoc.FirstBlock(blocks, newsdoc.BlocksWithType(coreText))
	if !ok {
		t.Fatal("expected to find a core/text block")
	}

	if b.Role != "heading" {
		t.Errorf("expected first core/text to be heading, got %q", b.Role)
	}
}

func TestFirstBlockNotFound(t *testing.T) {
	blocks := sampleBlocks()

	_, ok := newsdoc.FirstBlock(blocks, newsdoc.BlocksWithType("core/video"))
	if ok {
		t.Error("should not find non-existent type")
	}
}

func TestFirstBlockEmptyList(t *testing.T) {
	_, ok := newsdoc.FirstBlock(nil, newsdoc.BlocksWithType(coreText))
	if ok {
		t.Error("should not find anything in nil list")
	}
}

func TestAllBlocks(t *testing.T) {
	blocks := sampleBlocks()

	results := newsdoc.AllBlocks(blocks, newsdoc.BlocksWithType(coreText))
	if len(results) != 3 {
		t.Fatalf("expected 3 core/text blocks, got %d", len(results))
	}
}

func TestAllBlocksNoMatch(t *testing.T) {
	blocks := sampleBlocks()

	results := newsdoc.AllBlocks(blocks, newsdoc.BlocksWithType("core/video"))
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestAllBlocksEmptyList(t *testing.T) {
	results := newsdoc.AllBlocks(nil, newsdoc.BlocksWithType(coreText))
	if len(results) != 0 {
		t.Errorf("expected no results from nil list, got %d", len(results))
	}
}

func TestDropBlocks(t *testing.T) {
	blocks := sampleBlocks()

	result := newsdoc.DropBlocks(blocks, newsdoc.BlocksWithType(coreText))
	if len(result) != 2 {
		t.Fatalf("expected 2 blocks remaining, got %d", len(result))
	}

	for _, b := range result {
		if b.Type == coreText {
			t.Error("core/text blocks should have been dropped")
		}
	}
}

func TestDropBlocksNoMatch(t *testing.T) {
	blocks := sampleBlocks()

	result := newsdoc.DropBlocks(blocks, newsdoc.BlocksWithType("core/video"))
	if len(result) != 5 {
		t.Errorf("expected all 5 blocks to remain, got %d", len(result))
	}
}

func TestDedupeBlocks(t *testing.T) {
	blocks := sampleBlocks()
	matcher := newsdoc.BlocksWithType(coreText)

	result := newsdoc.DedupeBlocks(blocks, matcher)

	textCount := 0

	for _, b := range result {
		if b.Type == coreText {
			textCount++
		}
	}

	if textCount != 1 {
		t.Errorf("expected exactly 1 core/text block after dedup, got %d", textCount)
	}

	// The first core/text block (heading) should be the one kept.
	first, ok := newsdoc.FirstBlock(result, matcher)
	if !ok {
		t.Fatal("expected to find a core/text block")
	}

	if first.Role != "heading" {
		t.Errorf("expected the first core/text block to be kept, got role %q", first.Role)
	}
}

func TestDedupeBlocksNoMatch(t *testing.T) {
	blocks := sampleBlocks()

	result := newsdoc.DedupeBlocks(blocks, newsdoc.BlocksWithType("core/video"))
	if len(result) != 5 {
		t.Errorf("expected all blocks to remain when nothing matches, got %d", len(result))
	}
}

func TestDedupeBlocksSingleMatch(t *testing.T) {
	blocks := sampleBlocks()

	result := newsdoc.DedupeBlocks(blocks, newsdoc.BlocksWithType("core/image"))
	if len(result) != 5 {
		t.Errorf("expected all blocks to remain when only one matches, got %d", len(result))
	}
}

func TestAlterBlocks(t *testing.T) {
	blocks := sampleBlocks()

	newsdoc.AlterBlocks(blocks, newsdoc.BlocksWithType(coreText), func(b *newsdoc.Block) {
		b.Name = altered
	})

	for _, b := range blocks {
		if b.Type == coreText && b.Name != altered {
			t.Errorf("core/text block should have been altered, got Name=%q", b.Name)
		}

		if b.Type != coreText && b.Name == altered {
			t.Error("non-matching block should not have been altered")
		}
	}
}

func TestAlterBlocksNoMatch(t *testing.T) {
	blocks := sampleBlocks()

	// Should not panic.
	newsdoc.AlterBlocks(blocks, newsdoc.BlocksWithType("core/video"), func(b *newsdoc.Block) {
		b.Name = altered
	})

	for _, b := range blocks {
		if b.Name == altered {
			t.Error("no blocks should have been altered")
		}
	}
}

func TestAlterFirstBlock(t *testing.T) {
	blocks := sampleBlocks()

	newsdoc.AlterFirstBlock(blocks, newsdoc.BlocksWithType(coreText), func(b *newsdoc.Block) {
		b.Name = firstAltered
	})

	if blocks[0].Name != firstAltered {
		t.Error("first matching block should have been altered")
	}

	// Second core/text should NOT be altered.
	if blocks[1].Name == firstAltered {
		t.Error("second matching block should not have been altered")
	}
}

func TestAlterFirstBlockNoMatch(t *testing.T) {
	blocks := sampleBlocks()

	newsdoc.AlterFirstBlock(blocks, newsdoc.BlocksWithType("core/video"), func(b *newsdoc.Block) {
		b.Name = altered
	})

	for _, b := range blocks {
		if b.Name == altered {
			t.Error("no blocks should have been altered when none match")
		}
	}
}

func TestUpsertBlockUpdate(t *testing.T) {
	blocks := sampleBlocks()

	result := newsdoc.UpsertBlock(
		blocks,
		newsdoc.BlocksWithType("core/image"),
		newsdoc.Block{Type: "core/image"},
		func(b newsdoc.Block) newsdoc.Block {
			b.Title = "Updated Image"

			return b
		},
	)

	if len(result) != len(blocks) {
		t.Errorf("expected same length after update, got %d", len(result))
	}

	img, ok := newsdoc.FirstBlock(result, newsdoc.BlocksWithType("core/image"))
	if !ok {
		t.Fatal("expected to find core/image block")
	}

	if img.Title != "Updated Image" {
		t.Errorf("expected updated title, got %q", img.Title)
	}
}

func TestUpsertBlockInsert(t *testing.T) {
	blocks := sampleBlocks()
	originalLen := len(blocks)

	result := newsdoc.UpsertBlock(
		blocks,
		newsdoc.BlocksWithType("core/video"),
		newsdoc.Block{Type: "core/video"},
		func(b newsdoc.Block) newsdoc.Block {
			b.Title = newVideo

			return b
		},
	)

	if len(result) != originalLen+1 {
		t.Errorf("expected length %d after insert, got %d", originalLen+1, len(result))
	}

	video, ok := newsdoc.FirstBlock(result, newsdoc.BlocksWithType("core/video"))
	if !ok {
		t.Fatal("expected to find inserted core/video block")
	}

	if video.Title != newVideo {
		t.Errorf("expected title 'New Video', got %q", video.Title)
	}
}

func TestWithBlockOfTypeUpdate(t *testing.T) {
	blocks := []newsdoc.Block{
		{Type: "core/newsvalue", Data: newsdoc.DataMap{"score": "3"}},
	}

	result := newsdoc.WithBlockOfType(blocks, "core/newsvalue", func(b newsdoc.Block) newsdoc.Block {
		b.Data["score"] = "5"

		return b
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 block, got %d", len(result))
	}

	if result[0].Data["score"] != "5" {
		t.Errorf("expected updated score, got %q", result[0].Data["score"])
	}
}

func TestWithBlockOfTypeInsert(t *testing.T) {
	var blocks []newsdoc.Block

	result := newsdoc.WithBlockOfType(blocks, "core/newsvalue", func(b newsdoc.Block) newsdoc.Block {
		b.Data = newsdoc.DataMap{"score": "5"}

		return b
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 block, got %d", len(result))
	}

	if result[0].Type != "core/newsvalue" {
		t.Errorf("expected type 'core/newsvalue', got %q", result[0].Type)
	}

	if result[0].Data["score"] != "5" {
		t.Errorf("expected score '5', got %q", result[0].Data["score"])
	}
}

func TestAddOrReplaceBlockReplace(t *testing.T) {
	blocks := sampleBlocks()

	replacement := newsdoc.Block{Type: "core/image", Title: "Replaced", Rel: "replaced"}

	result := newsdoc.AddOrReplaceBlock(blocks, newsdoc.BlocksWithType("core/image"), replacement)

	if len(result) != len(blocks) {
		t.Errorf("expected same length after replace, got %d", len(result))
	}

	img, ok := newsdoc.FirstBlock(result, newsdoc.BlocksWithType("core/image"))
	if !ok {
		t.Fatal("expected to find core/image block")
	}

	if img.Title != "Replaced" {
		t.Errorf("expected replaced title, got %q", img.Title)
	}

	// Original rel should be gone since the whole block was replaced.
	if img.Rel != "replaced" {
		t.Errorf("expected replaced rel, got %q", img.Rel)
	}
}

func TestAddOrReplaceBlockAdd(t *testing.T) {
	blocks := sampleBlocks()
	originalLen := len(blocks)

	newBlock := newsdoc.Block{Type: "core/video", Title: newVideo}

	result := newsdoc.AddOrReplaceBlock(blocks, newsdoc.BlocksWithType("core/video"), newBlock)

	if len(result) != originalLen+1 {
		t.Errorf("expected length %d after add, got %d", originalLen+1, len(result))
	}

	video, ok := newsdoc.FirstBlock(result, newsdoc.BlocksWithType("core/video"))
	if !ok {
		t.Fatal("expected to find added core/video block")
	}

	if video.Title != newVideo {
		t.Errorf("expected title 'New Video', got %q", video.Title)
	}
}
