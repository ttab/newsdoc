package newsdoc

import (
	"slices"
)

// FirstBlock returns the first block matching the selector.
func FirstBlock(list []Block, selector BlockMatcher) (Block, bool) {
	for i := range list {
		if !selector.Match(list[i]) {
			continue
		}

		return list[i], true
	}

	return Block{}, false
}

// DropBlocks removes all blocks matching the selector.
func DropBlocks(list []Block, selector BlockMatcher) []Block {
	return slices.DeleteFunc(list, selector.Match)
}

// DedupeBlocks removes all but the first block matching the selector.
func DedupeBlocks(list []Block, selector BlockMatcher) []Block {
	var (
		found bool
		keep  []Block
	)

	for _, b := range list {
		match := selector.Match(b)
		if match && found {
			continue
		}

		found = found || match

		keep = append(keep, b)
	}

	return keep
}

// AlterBlocks calls fn for each block matching the selector.
func AlterBlocks(list []Block, selector BlockMatcher, fn func(*Block)) {
	for i := range list {
		if !selector.Match(list[i]) {
			continue
		}

		fn(&list[i])
	}
}

// UpsertBlock inserts a new block or updates an existing block if it matches
// the selector. The function fn will be called on the inserted or existing
// block.
func UpsertBlock(
	list []Block, selector BlockMatcher, insert Block,
	fn func(b Block) Block,
) []Block {
	for i := range list {
		if !selector.Match(list[i]) {
			continue
		}

		list[i] = fn(list[i])

		return list
	}

	list = append(list, fn(insert))

	return list
}

// WithBlockOfType upserts a block with the given type, see UpsertBlock().
func WithBlockOfType(
	list []Block, blockType string, fn func(b Block) Block,
) []Block {
	return UpsertBlock(
		list,
		BlocksWithType(blockType),
		Block{Type: blockType},
		fn,
	)
}

// AddOrReplaceBlock inserts a new block into the list, or replaces the first
// block matching the selector.
func AddOrReplaceBlock(
	list []Block, selector BlockMatcher, insert Block,
) []Block {
	return UpsertBlock(list, selector, insert, func(_ Block) Block {
		return insert
	})
}
