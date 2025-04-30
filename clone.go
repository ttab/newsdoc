package newsdoc

import (
	"maps"
)

// Clone returns a deep copy of the document.
func (d Document) Clone() Document {
	c := d

	c.Meta = cloneBlocks(d.Meta)
	c.Links = cloneBlocks(d.Links)
	c.Content = cloneBlocks(d.Content)

	return c
}

func cloneBlocks(blocks []Block) []Block {
	c := make([]Block, len(blocks))

	for i := range blocks {
		c[i] = blocks[i].Clone()
	}

	return c
}

// Clone returns a deep copy of the block.
func (b Block) Clone() Block {
	c := b

	c.Meta = cloneBlocks(b.Meta)
	c.Links = cloneBlocks(b.Links)
	c.Content = cloneBlocks(b.Content)
	c.Data = maps.Clone(b.Data)

	return c
}
