package newsdoc

// BlockMatcher checks if a block matches a condition.
type BlockMatcher interface {
	// Match returns true if the block matches the condition.
	Match(block Block) bool
}

// BlockMatchFunc is a custom BlockMatcher function.
type BlockMatchFunc func(block Block) bool

// Implements BlockMatcher.
func (fn BlockMatchFunc) Match(block Block) bool {
	return fn(block)
}

// BlockRole can be used to check that a block has a specific role.
type BlockRole string

// Implements BlockMatcher.
func (role BlockRole) Match(block Block) bool {
	return block.Role == string(role)
}

// BlockMatchesAll returns a block matcher that returns true if a block matches
// all the conditions.
func BlockMatchesAll(matchers ...BlockMatcher) BlockMatcher {
	return BlockMatchFunc(func(block Block) bool {
		for _, m := range matchers {
			if !m.Match(block) {
				return false
			}
		}

		return true
	})
}

// BlockMatchesAny returns a block matcher that returns true if a block matches
// any of the conditions.
func BlocksMatchesAny(matchers ...BlockMatcher) BlockMatcher {
	return BlockMatchFunc(func(block Block) bool {
		for _, m := range matchers {
			if m.Match(block) {
				return true
			}
		}

		return false
	})
}

// BlockDoesntMatch returns a block matcher that negates the selector.
func BlockDoesntMatch(selector BlockMatcher) BlockMatcher {
	return BlockMatchFunc(func(block Block) bool {
		return !selector.Match(block)
	})
}

// BlocksWithType returns a BlockMatcher that matches blocks with the given
// type.
func BlocksWithType(blockType string) BlockMatcher {
	return blockSelector{
		bType: &blockType,
	}
}

// BlocksWithRel returns a BlockMatcher that matches blocks with the given rel.
func BlocksWithRel(rel string) BlockMatcher {
	return blockSelector{
		rel: &rel,
	}
}

// BlocksWithTypeAndRel returns a BlockMatcher that matches blocks with the
// given type and rel.
func BlocksWithTypeAndRel(blockType string, rel string) BlockMatcher {
	return blockSelector{
		bType: &blockType,
		rel:   &rel,
	}
}

// BlocksWithTypeAndRole returns a BlockMatcher that matches blocks with the
// given type and role.
func BlocksWithTypeAndRole(blockType string, role string) BlockMatcher {
	return blockSelector{
		bType: &blockType,
		role:  &role,
	}
}

type blockSelector struct {
	bType *string
	rel   *string
	role  *string
}

func (sel blockSelector) Match(block Block) bool {
	if sel.bType != nil && *sel.bType != block.Type {
		return false
	}

	if sel.rel != nil && *sel.rel != block.Rel {
		return false
	}

	if sel.role != nil && *sel.role != block.Role {
		return false
	}

	return true
}
