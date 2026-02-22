package newsdoc

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"slices"
)

type ValueExtractor struct {
	Selectors      []BlockSelector
	ChildSelectors []BlockSelector `json:",omitempty"`
	ValueKind      ValueKind
	Values         []ValueSpec
}

var (
	dataPrefix  = []byte(".data{")
	attrPrefix  = []byte("@{")
	valuesEnd   = []byte("}")
	bPeriod     = []byte(".")
	bStartParen = []byte("(")
	bEndParen   = []byte(")")
	bEqual      = []byte("=")
	bComma      = []byte(",")
	bColon      = []byte(":")
	bQMark      = []byte("?")
	bDataDot    = []byte("data.")
	bQuote      = byte('\'')
	bBackslash  = byte('\\')
)

// findClosingQuote returns the index of the closing single quote in s,
// skipping escaped quotes (\'). The input s must start after the opening
// quote. Returns -1 if no unescaped closing quote is found.
func findClosingQuote(s []byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == bBackslash && i+1 < len(s) {
			i++

			continue
		}

		if s[i] == bQuote {
			return i
		}
	}

	return -1
}

// unescapeQuoted removes backslash escapes from a quoted value, turning \' into
// ' and \\ into \.
func unescapeQuoted(s []byte) []byte {
	if !bytes.ContainsRune(s, '\\') {
		return s
	}

	out := make([]byte, 0, len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == bBackslash && i+1 < len(s) {
			i++
		}

		out = append(out, s[i])
	}

	return out
}

// indexByteOutsideQuotes returns the index of the first occurrence of c in s
// that is not inside a single-quoted string, or -1 if not found.
func indexByteOutsideQuotes(s []byte, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == bQuote {
			end := findClosingQuote(s[i+1:])
			if end != -1 {
				i += end + 1
			}

			continue
		}

		if s[i] == c {
			return i
		}
	}

	return -1
}

// lastIndexOutsideQuotes returns the index of the last occurrence of sep in s
// that starts outside a single-quoted string, or -1 if not found.
func lastIndexOutsideQuotes(s, sep []byte) int {
	last := -1

	for i := 0; i < len(s); i++ {
		if s[i] == bQuote {
			end := findClosingQuote(s[i+1:])
			if end != -1 {
				i += end + 1
			}

			continue
		}

		if bytes.HasPrefix(s[i:], sep) {
			last = i
		}
	}

	return last
}

func ValueExtractorFromString(text string) (*ValueExtractor, error) {
	return ValueExtractorFromBytes([]byte(text))
}

func ValueExtractorFromBytes(text []byte) (*ValueExtractor, error) {
	var ve ValueExtractor

	// Find the split point between selectors and value spec, ignoring
	// occurrences inside single-quoted attribute values.
	dataIdx := lastIndexOutsideQuotes(text, dataPrefix)
	attrIdx := lastIndexOutsideQuotes(text, attrPrefix)

	var (
		valueSpecInner []byte
		selector       []byte
	)

	switch {
	case dataIdx != -1 && attrIdx != -1:
		ve.ValueKind = ValueKindCombined
		selector = text[:min(attrIdx, dataIdx)]

		if len(selector) == 0 {
			return nil, fmt.Errorf(
				"combined extraction requires at least one selector")
		}

		attrInner, dataInner, err := parseCombinedSpecs(text, attrIdx, dataIdx)
		if err != nil {
			return nil, err
		}

		attrValues, err := parseValues(attrInner)
		if err != nil {
			return nil, fmt.Errorf("attribute values: %w", err)
		}

		for i := range attrValues {
			attrValues[i].Source = ValueSourceAttributes
		}

		dataValues, err := parseValues(dataInner)
		if err != nil {
			return nil, fmt.Errorf("data values: %w", err)
		}

		for i := range dataValues {
			dataValues[i].Source = ValueSourceData
		}

		ve.Values = slices.Concat(attrValues, dataValues)
	case dataIdx != -1:
		ve.ValueKind = ValueKindData
		selector = text[:dataIdx]
		valueSpecInner, _ = bytes.CutPrefix(text[dataIdx:], dataPrefix)

		if dataIdx == 0 {
			return nil, fmt.Errorf("documents do not have data blocks")
		}
	case attrIdx != -1:
		ve.ValueKind = ValueKindAttributes
		selector = text[:attrIdx]
		valueSpecInner, _ = bytes.CutPrefix(text[attrIdx:], attrPrefix)
	default:
		ve.ValueKind = ValueKindBlock

		nameBytes, rest, hasName := bytes.Cut(text, bEqual)
		if !hasName {
			return nil, fmt.Errorf(
				"block extraction requires a name prefix (name=.selectors)")
		}

		name := string(bytes.TrimSpace(nameBytes))
		if name == "" {
			return nil, fmt.Errorf("block extraction name cannot be empty")
		}

		var annotation string

		// Look for an annotation suffix after the last closing paren
		// to avoid matching ':' inside quoted attribute values.
		afterLastParen := rest
		if lp := bytes.LastIndex(rest, bEndParen); lp != -1 {
			afterLastParen = rest[lp+1:]
		}

		if ci := bytes.Index(afterLastParen, bColon); ci != -1 {
			offset := len(rest) - len(afterLastParen) + ci
			annotation = string(bytes.TrimSpace(rest[offset+1:]))
			rest = rest[:offset]
		}

		selector = rest

		ve.Values = []ValueSpec{{
			Name:       name,
			Annotation: annotation,
		}}
	}

	// Split parent and child selectors on '#', skipping '#' inside
	// single-quoted attribute values.
	var childSelector []byte

	if hashIdx := indexByteOutsideQuotes(selector, '#'); hashIdx != -1 {
		childSelector = selector[hashIdx+1:]
		selector = selector[:hashIdx]
	}

	selectors, err := parseSelectors(selector)
	if err != nil {
		return nil, err
	}

	ve.Selectors = selectors

	if len(childSelector) > 0 {
		childSelectors, err := parseSelectors(childSelector)
		if err != nil {
			return nil, fmt.Errorf("child selectors: %w", err)
		}

		if len(childSelectors) == 0 {
			return nil, fmt.Errorf("empty child selector after '#'")
		}

		ve.ChildSelectors = childSelectors
	}

	if ve.ValueKind == ValueKindBlock {
		if len(ve.Selectors) == 0 {
			return nil, fmt.Errorf(
				"block extraction requires at least one selector")
		}

		return &ve, nil
	}

	// Combined expressions have their values parsed in the switch above.
	if ve.ValueKind == ValueKindCombined {
		return &ve, nil
	}

	if !bytes.HasSuffix(valueSpecInner, valuesEnd) {
		return nil, fmt.Errorf("invalid format: expected '}' at end of value specifier")
	}

	valueSpecInner = valueSpecInner[:len(valueSpecInner)-1]

	values, err := parseValues(valueSpecInner)
	if err != nil {
		return nil, err
	}

	ve.Values = values

	return &ve, nil
}

func (ve *ValueExtractor) Collect(doc Document) []ExtractedItems {
	// If we don't have a selector the value extraction targets the document
	// itself.
	if len(ve.Selectors) == 0 {
		docValues := extractDocumentAttributes(doc, ve.Values)
		if len(docValues) == 0 {
			return nil
		}

		return []ExtractedItems{docValues}
	}

	var blocks []Block

	root := ve.Selectors[0]

	switch root.Kind {
	case BlockKindContent:
		blocks = doc.Content
	case BlockKindLinks:
		blocks = doc.Links
	case BlockKindMeta:
		blocks = doc.Meta
	default:
		panic(fmt.Sprintf("unexpected newsdoc.BlockKind: %#v", root.Kind))
	}

	matches := root.Iterator(slices.Values(blocks))

	for i := 1; i < len(ve.Selectors); i++ {
		sel := ve.Selectors[i]

		var srcBlocks []iter.Seq[Block]

		for b := range matches {
			switch sel.Kind {
			case BlockKindContent:
				srcBlocks = append(srcBlocks, slices.Values(b.Content))
			case BlockKindLinks:
				srcBlocks = append(srcBlocks, slices.Values(b.Links))
			case BlockKindMeta:
				srcBlocks = append(srcBlocks, slices.Values(b.Meta))
			default:
				panic(fmt.Sprintf("unexpected newsdoc.BlockKind: %#v", root.Kind))
			}
		}

		matches = sel.Iterator(concatIter(srcBlocks...))
	}

	if len(ve.ChildSelectors) > 0 {
		childSelectors := ve.ChildSelectors
		prev := matches

		matches = func(yield func(Block) bool) {
			for b := range prev {
				if !hasMatchingChildren(b, childSelectors) {
					continue
				}

				if !yield(b) {
					return
				}
			}
		}
	}

	var extracts []ExtractedItems

	if ve.ValueKind == ValueKindBlock {
		spec := ve.Values[0]

		for b := range matches {
			block := b

			extracts = append(extracts, ExtractedItems{
				spec.Name: {
					Name:       spec.Name,
					Block:      &block,
					Annotation: spec.Annotation,
				},
			})
		}

		return extracts
	}

	if ve.ValueKind == ValueKindCombined {
		for b := range matches {
			e := extractCombinedItems(b, ve.Values)
			if len(e) == 0 {
				continue
			}

			extracts = append(extracts, e)
		}

		return extracts
	}

	var accessor func(b Block, name string) string

	switch ve.ValueKind {
	case ValueKindAttributes:
		accessor = getBlockAttribute
	case ValueKindData:
		accessor = getBlockData
	case ValueKindBlock:
		panic("ValueKindBlock should have been handled above")
	case ValueKindCombined:
		panic("ValueKindCombined should have been handled above")
	}

	for b := range matches {
		e := extractItems(b, ve.Values, accessor)
		if len(e) == 0 {
			continue
		}

		extracts = append(extracts, e)
	}

	return extracts
}

func extractDocumentAttributes(doc Document, spec []ValueSpec) ExtractedItems {
	e := make(ExtractedItems)

	for _, v := range spec {
		value := getDocumentAttribute(doc, v.Name)
		switch {
		case value == "" && !v.Optional:
			return nil
		case value == "" && v.Optional:
			continue
		}

		ev := ExtractedValue{
			Name:       v.Name,
			Value:      value,
			Annotation: v.Annotation,
			Role:       v.Role,
		}

		e[v.Name] = ev
	}

	return e
}

type documentAttributeKey string

const (
	docAttrType     documentAttributeKey = "type"
	docAttrLanguage documentAttributeKey = "language"
	docAttrTitle    documentAttributeKey = "title"
	docAttrUUID     documentAttributeKey = "uuid"
	docAttrURI      documentAttributeKey = "uri"
	docAttrURL      documentAttributeKey = "url"
)

func getDocumentAttribute(doc Document, name string) string {
	switch documentAttributeKey(name) {
	case docAttrUUID:
		return doc.UUID
	case docAttrType:
		return doc.Type
	case docAttrURI:
		return doc.URI
	case docAttrURL:
		return doc.URL
	case docAttrTitle:
		return doc.Title
	case docAttrLanguage:
		return doc.Language
	}

	return ""
}

func extractItems(
	b Block,
	spec []ValueSpec,
	accessor func(b Block, name string) string,
) ExtractedItems {
	e := make(ExtractedItems)

	for _, v := range spec {
		value := accessor(b, v.Name)
		switch {
		case value == "" && !v.Optional:
			return nil
		case value == "" && v.Optional:
			continue
		}

		ev := ExtractedValue{
			Name:       v.Name,
			Value:      value,
			Annotation: v.Annotation,
			Role:       v.Role,
		}

		e[v.Name] = ev
	}

	return e
}

func extractCombinedItems(b Block, spec []ValueSpec) ExtractedItems {
	e := make(ExtractedItems)

	for _, v := range spec {
		var value string

		switch v.Source {
		case ValueSourceAttributes:
			value = getBlockAttribute(b, v.Name)
		case ValueSourceData:
			value = getBlockData(b, v.Name)
		}

		switch {
		case value == "" && !v.Optional:
			return nil
		case value == "" && v.Optional:
			continue
		}

		e[v.Name] = ExtractedValue{
			Name:       v.Name,
			Value:      value,
			Annotation: v.Annotation,
			Role:       v.Role,
		}
	}

	return e
}

func getBlockData(b Block, name string) string {
	return b.Data.Get(name, "")
}

type blockAttributeKey string

const (
	blockAttrID          blockAttributeKey = "id"
	blockAttrUUID        blockAttributeKey = "uuid"
	blockAttrType        blockAttributeKey = "type"
	blockAttrURI         blockAttributeKey = "uri"
	blockAttrURL         blockAttributeKey = "url"
	blockAttrTitle       blockAttributeKey = "title"
	blockAttrRel         blockAttributeKey = "rel"
	blockAttrName        blockAttributeKey = "name"
	blockAttrValue       blockAttributeKey = "value"
	blockAttrContentType blockAttributeKey = "contenttype"
	blockAttrRole        blockAttributeKey = "role"
	blockAttrSensitivity blockAttributeKey = "sensitivity"
)

func getBlockAttribute(block Block, name string) string {
	switch blockAttributeKey(name) {
	case blockAttrUUID:
		return block.UUID
	case blockAttrID:
		return block.ID
	case blockAttrType:
		return block.Type
	case blockAttrURI:
		return block.URI
	case blockAttrURL:
		return block.URL
	case blockAttrTitle:
		return block.Title
	case blockAttrRel:
		return block.Rel
	case blockAttrName:
		return block.Name
	case blockAttrValue:
		return block.Value
	case blockAttrContentType:
		return block.Contenttype
	case blockAttrRole:
		return block.Role
	case blockAttrSensitivity:
		return block.Sensitivity
	}

	return ""
}

type ValueSpec struct {
	Name       string
	Source     ValueSource `json:",omitempty"`
	Optional   bool        `json:",omitempty"`
	Annotation string      `json:",omitempty"`
	Role       string      `json:",omitempty"`
}

type ExtractedItems map[string]ExtractedValue

type ExtractedValue struct {
	Name       string
	Value      string `json:",omitempty"`
	Block      *Block `json:",omitempty"`
	Annotation string `json:",omitempty"`
	Role       string `json:",omitempty"`
}

type BlockKind string

const (
	BlockKindMeta    BlockKind = "meta"
	BlockKindLinks   BlockKind = "links"
	BlockKindContent BlockKind = "content"
)

// ValueKind describes what kind of values a ValueExtractor produces.
type ValueKind string

const (
	// ValueKindAttributes extracts block attribute values using @{}.
	ValueKindAttributes ValueKind = "attributes"
	// ValueKindData extracts block data map values using .data{}.
	ValueKindData ValueKind = "data"
	// ValueKindBlock extracts matched blocks themselves (name=.selectors).
	ValueKindBlock ValueKind = "block"
	// ValueKindCombined extracts both attribute and data values from matched
	// blocks using @{}.data{} in a single expression.
	ValueKindCombined ValueKind = "combined"
)

// ValueSource identifies whether a value spec in a combined extraction targets
// block attributes or block data. It is only populated for ValueKindCombined
// expressions.
type ValueSource string

const (
	ValueSourceData       ValueSource = "data"
	ValueSourceAttributes ValueSource = "attributes"
)

// DataFilterMode describes the comparison mode for a data filter.
type DataFilterMode string

const (
	// DataFilterExact matches when the data key exists with the exact
	// value.
	DataFilterExact DataFilterMode = "exact"
	// DataFilterExists matches when the data key exists, even if empty.
	DataFilterExists DataFilterMode = "exists"
	// DataFilterNonEmpty matches when the data key exists and is non-empty.
	DataFilterNonEmpty DataFilterMode = "non-empty"
)

// DataFilter is a filter condition on a block's data map.
type DataFilter struct {
	Key   string
	Value string `json:",omitempty"`
	Mode  DataFilterMode
}

// matches reports whether the data filter matches the given block.
func (df DataFilter) matches(b Block) bool {
	switch df.Mode {
	case DataFilterExact:
		return b.Data.Get(df.Key, "") == df.Value
	case DataFilterExists:
		_, ok := b.Data[df.Key]

		return ok
	case DataFilterNonEmpty:
		return b.Data.Get(df.Key, "") != ""
	}

	return false
}

// FilterOp is the boolean operator for a filter node.
type FilterOp string

const (
	// FilterOpAnd combines children with logical AND.
	FilterOpAnd FilterOp = "and"
	// FilterOpOr combines children with logical OR.
	FilterOpOr FilterOp = "or"
)

// FilterNode is a node in a boolean filter expression tree. Branch nodes have
// Op and Children set; leaf nodes have either Attr+Value (attribute match) or
// Data (data filter) set.
type FilterNode struct {
	Op       FilterOp     `json:",omitempty"`
	Children []FilterNode `json:",omitempty"`
	Attr     string       `json:",omitempty"`
	Value    string       `json:",omitempty"`
	Data     *DataFilter  `json:",omitempty"`
}

// Matches reports whether the filter node matches the given block. A nil node
// matches all blocks.
func (fn *FilterNode) Matches(b Block) bool {
	if fn == nil {
		return true
	}

	switch fn.Op {
	case FilterOpAnd:
		for _, child := range fn.Children {
			if !child.Matches(b) {
				return false
			}
		}

		return true
	case FilterOpOr:
		for _, child := range fn.Children {
			if child.Matches(b) {
				return true
			}
		}

		return false
	default:
		// Leaf node.
		if fn.Data != nil {
			return fn.Data.matches(b)
		}

		return getBlockAttribute(b, fn.Attr) == fn.Value
	}
}

// BlockSelector selects blocks by kind and optional attribute/data filters.
type BlockSelector struct {
	Kind   BlockKind
	Filter *FilterNode `json:",omitempty"`
}

func (bs BlockSelector) Iterator(blocks iter.Seq[Block]) iter.Seq[Block] {
	return func(yield func(Block) bool) {
		for b := range blocks {
			if !bs.Matches(b) {
				continue
			}

			if !yield(b) {
				return
			}
		}
	}
}

// concatIter returns an iterator over the concatenation of the sequences.
func concatIter[V any](seqs ...iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, seq := range seqs {
			for e := range seq {
				if !yield(e) {
					return
				}
			}
		}
	}
}

// hasMatchingChildren checks if a block has descendants matching the given
// child selector chain.
func hasMatchingChildren(b Block, selectors []BlockSelector) bool {
	if len(selectors) == 0 {
		return true
	}

	first := selectors[0]

	var blocks []Block

	switch first.Kind {
	case BlockKindContent:
		blocks = b.Content
	case BlockKindLinks:
		blocks = b.Links
	case BlockKindMeta:
		blocks = b.Meta
	}

	matches := first.Iterator(slices.Values(blocks))

	for i := 1; i < len(selectors); i++ {
		sel := selectors[i]

		var srcBlocks []iter.Seq[Block]

		for m := range matches {
			switch sel.Kind {
			case BlockKindContent:
				srcBlocks = append(srcBlocks, slices.Values(m.Content))
			case BlockKindLinks:
				srcBlocks = append(srcBlocks, slices.Values(m.Links))
			case BlockKindMeta:
				srcBlocks = append(srcBlocks, slices.Values(m.Meta))
			}
		}

		matches = sel.Iterator(concatIter(srcBlocks...))
	}

	for range matches {
		return true
	}

	return false
}

func (bs BlockSelector) FilterBlocks(blocks []Block) []Block {
	return slices.Collect(bs.Iterator(slices.Values(blocks)))
}

func (bs BlockSelector) Matches(b Block) bool {
	return bs.Filter.Matches(b)
}

var validAttributeKeys = map[string]struct{}{
	"id":          {},
	"uuid":        {},
	"uri":         {},
	"url":         {},
	"type":        {},
	"rel":         {},
	"role":        {},
	"name":        {},
	"value":       {},
	"contenttype": {},
	"sensitivity": {},
}

// validateAttributeKey checks that key is a known block attribute.
func validateAttributeKey(key string) error {
	if _, ok := validAttributeKeys[key]; !ok {
		return fmt.Errorf("unknown attribute key: %s", key)
	}

	return nil
}

// attrParser is a recursive descent parser for attribute filter expressions.
type attrParser struct {
	input []byte
	pos   int
}

func (p *attrParser) skipSpace() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *attrParser) atEnd() bool {
	return p.pos >= len(p.input)
}

func (p *attrParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}

	return p.input[p.pos]
}

// isOrKeyword reports whether the input at the current position is the "or"
// keyword. We require a word boundary (space, ')' or end of input) after "or"
// to avoid matching keys like "order".
func (p *attrParser) isOrKeyword() bool {
	if p.pos+2 > len(p.input) {
		return false
	}

	if p.input[p.pos] != 'o' || p.input[p.pos+1] != 'r' {
		return false
	}

	if p.pos+2 == len(p.input) {
		return true
	}

	c := p.input[p.pos+2]

	return c == ' ' || c == ')'
}

// parseOrExpr parses: or_expr = and_expr ('or' and_expr)*.
func (p *attrParser) parseOrExpr() (FilterNode, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return FilterNode{}, err
	}

	children := []FilterNode{left}

	for {
		p.skipSpace()

		if !p.isOrKeyword() {
			break
		}

		p.pos += 2 // consume "or"

		p.skipSpace()

		if p.atEnd() || p.peek() == ')' {
			return FilterNode{}, fmt.Errorf(
				"unexpected end after 'or' at position %d", p.pos)
		}

		child, err := p.parseAndExpr()
		if err != nil {
			return FilterNode{}, err
		}

		children = append(children, child)
	}

	if len(children) == 1 {
		return children[0], nil
	}

	return FilterNode{
		Op:       FilterOpOr,
		Children: children,
	}, nil
}

// parseAndExpr parses: and_expr = factor (factor)*. Implicit AND via
// space-separated factors.
func (p *attrParser) parseAndExpr() (FilterNode, error) {
	first, err := p.parseFactor()
	if err != nil {
		return FilterNode{}, err
	}

	children := []FilterNode{first}

	for {
		p.skipSpace()

		if p.atEnd() || p.peek() == ')' || p.isOrKeyword() {
			break
		}

		child, err := p.parseFactor()
		if err != nil {
			return FilterNode{}, err
		}

		children = append(children, child)
	}

	if len(children) == 1 {
		return children[0], nil
	}

	return FilterNode{
		Op:       FilterOpAnd,
		Children: children,
	}, nil
}

// parseFactor parses: factor = '(' or_expr ')' | atom.
func (p *attrParser) parseFactor() (FilterNode, error) {
	if p.atEnd() {
		return FilterNode{}, fmt.Errorf("unexpected end of attributes")
	}

	if p.isOrKeyword() {
		return FilterNode{}, fmt.Errorf(
			"unexpected 'or' at position %d", p.pos)
	}

	if p.peek() == ')' {
		return FilterNode{}, fmt.Errorf(
			"unexpected ')' at position %d", p.pos)
	}

	if p.peek() == '(' {
		p.pos++ // consume '('

		p.skipSpace()

		if p.atEnd() {
			return FilterNode{}, fmt.Errorf(
				"unexpected end after '(' at position %d", p.pos)
		}

		node, err := p.parseOrExpr()
		if err != nil {
			return FilterNode{}, err
		}

		p.skipSpace()

		if p.atEnd() || p.peek() != ')' {
			return FilterNode{}, fmt.Errorf(
				"expected ')' to close group at position %d", p.pos)
		}

		p.pos++ // consume ')'

		return node, nil
	}

	return p.parseAtom()
}

// parseAtom parses: atom = data_filter | attr_match.
func (p *attrParser) parseAtom() (FilterNode, error) {
	if bytes.HasPrefix(p.input[p.pos:], bDataDot) {
		df, n, err := parseDataFilter(p.input[p.pos:])
		if err != nil {
			return FilterNode{}, err
		}

		p.pos += n

		return FilterNode{Data: &df}, nil
	}

	return p.parseAttrMatch()
}

// parseAttrMatch parses: key '=' quoted_value.
func (p *attrParser) parseAttrMatch() (FilterNode, error) {
	rest := p.input[p.pos:]

	eqIdx := bytes.IndexByte(rest, '=')
	if eqIdx == -1 {
		return FilterNode{}, fmt.Errorf(
			"invalid attribute format, expected '=' in: %q", rest)
	}

	key := bytes.TrimSpace(rest[:eqIdx])

	if err := validateAttributeKey(string(key)); err != nil {
		return FilterNode{}, err
	}

	p.pos += eqIdx + 1 // past the '='

	// Skip leading space before the quoted value.
	p.skipSpace()

	if p.atEnd() {
		return FilterNode{}, fmt.Errorf(
			"missing value for attribute key: %q", key)
	}

	if p.peek() != '\'' {
		return FilterNode{}, fmt.Errorf(
			"attribute value must be quoted: %q", p.input[p.pos:])
	}

	p.pos++ // skip opening quote

	endQuote := findClosingQuote(p.input[p.pos:])
	if endQuote == -1 {
		return FilterNode{}, fmt.Errorf(
			"unterminated quoted value in: %q", p.input[p.pos-1:])
	}

	value := unescapeQuoted(p.input[p.pos : p.pos+endQuote])
	p.pos += endQuote + 1 // skip past closing quote

	return FilterNode{
		Attr:  string(key),
		Value: string(value),
	}, nil
}

// parseAttributes parses the attribute filter expression, e.g.:
//
//	"type='core/text' rel='item'"
//	"type='core/event' data.date?? data.status='confirmed'"
//	"type='core/thing' (value='a' or value='b')"
func parseAttributes(attrsStr []byte) (*FilterNode, error) {
	p := &attrParser{input: bytes.TrimSpace(attrsStr)}

	node, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}

	p.skipSpace()

	if !p.atEnd() {
		return nil, fmt.Errorf(
			"unexpected content after attributes: %q", p.input[p.pos:])
	}

	return &node, nil
}

// parseDataFilter parses a data filter token from the start of b. It returns
// the parsed filter and the number of bytes consumed. The input must start with
// "data.".
func parseDataFilter(b []byte) (DataFilter, int, error) {
	// Find the end of this token (next space, ')' or end of input).
	spaceIdx := bytes.IndexByte(b, ' ')
	parenIdx := bytes.IndexByte(b, ')')

	tokenEnd := len(b)

	if spaceIdx != -1 {
		tokenEnd = spaceIdx
	}

	if parenIdx != -1 && parenIdx < tokenEnd {
		tokenEnd = parenIdx
	}

	token := b[:tokenEnd]
	rest := token[len(bDataDot):]

	// Check for existence modes first (order matters: ?? before ?).
	if bytes.HasSuffix(rest, []byte("??")) {
		key := rest[:len(rest)-2]
		if len(key) == 0 {
			return DataFilter{}, 0, fmt.Errorf(
				"empty key in data filter: %q", token)
		}

		return DataFilter{
			Key:  string(key),
			Mode: DataFilterNonEmpty,
		}, tokenEnd, nil
	}

	if bytes.HasSuffix(rest, bQMark) {
		key := rest[:len(rest)-1]
		if len(key) == 0 {
			return DataFilter{}, 0, fmt.Errorf(
				"empty key in data filter: %q", token)
		}

		return DataFilter{
			Key:  string(key),
			Mode: DataFilterExists,
		}, tokenEnd, nil
	}

	// Exact match: data.key='value'. The token boundary might be wrong
	// since the value can contain spaces, so we re-parse from the original
	// input.
	eqIdx := bytes.IndexByte(rest, '=')
	if eqIdx == -1 {
		return DataFilter{}, 0, fmt.Errorf(
			"invalid data filter, expected '?', '??', or '=' in: %q", token)
	}

	key := rest[:eqIdx]
	if len(key) == 0 {
		return DataFilter{}, 0, fmt.Errorf(
			"empty key in data filter: %q", token)
	}

	// Parse the quoted value from the full remaining input starting after
	// the '='.
	valStart := len(bDataDot) + eqIdx + 1
	valPart := b[valStart:]

	if len(valPart) == 0 || valPart[0] != '\'' {
		return DataFilter{}, 0, fmt.Errorf(
			"data filter value must be quoted: %q", token)
	}

	endQuoteIndex := findClosingQuote(valPart[1:])
	if endQuoteIndex == -1 {
		return DataFilter{}, 0, fmt.Errorf(
			"unterminated quoted value in data filter: %q", b)
	}

	// endQuoteIndex is relative to valPart[1:].
	endQuoteIndex++

	value := unescapeQuoted(valPart[1:endQuoteIndex])
	consumed := valStart + endQuoteIndex + 1

	return DataFilter{
		Key:   string(key),
		Value: string(value),
		Mode:  DataFilterExact,
	}, consumed, nil
}

// splitSelectors splits a selector chain on periods that are outside of
// parentheses. The leading period must already be stripped.
func splitSelectors(s []byte) [][]byte {
	var parts [][]byte

	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		case '\'':
			// Skip quoted values so that periods inside quotes are
			// not treated as separators.
			end := findClosingQuote(s[i+1:])
			if end != -1 {
				i += end + 1
			}
		}
	}

	parts = append(parts, s[start:])

	return parts
}

// parseSelectors manually parses the selector chain, e.g.:
//
//	".meta(type='example/thing').links"
//	".meta(type='core/event' data.date??).data{date}"
func parseSelectors(s []byte) ([]BlockSelector, error) {
	if len(s) == 0 {
		return nil, nil
	}

	if !bytes.HasPrefix(s, bPeriod) {
		return nil, fmt.Errorf("selector chain must start with '.'")
	}

	parts := splitSelectors(s[1:])
	selectors := make([]BlockSelector, 0, len(parts))

	for _, part := range parts {
		if len(part) == 0 {
			return nil, fmt.Errorf("empty selector part found (double dot '..')")
		}

		kindStr, attrsStr, foundParen := bytes.Cut(part, bStartParen)

		var selector BlockSelector

		// Set and validate BlockKind.
		switch BlockKind(kindStr) {
		case BlockKindMeta, BlockKindLinks, BlockKindContent:
			selector.Kind = BlockKind(kindStr)
		default:
			return nil, fmt.Errorf("unknown block kind: %s", kindStr)
		}

		// If there are parentheses, parse the attributes inside them.
		if foundParen {
			if !bytes.HasSuffix(attrsStr, bEndParen) {
				return nil, fmt.Errorf("mismatched parenthesis in selector: %q", part)
			}

			// Remove the trailing ')' before parsing.
			attrsStr = bytes.TrimSpace(attrsStr[:len(attrsStr)-1])

			if len(attrsStr) > 0 {
				filter, err := parseAttributes(attrsStr)
				if err != nil {
					return nil, err
				}

				selector.Filter = filter
			}
		}

		selectors = append(selectors, selector)
	}

	return selectors, nil
}

// parseCombinedSpecs extracts the inner byte slices for the @{} and .data{}
// specifiers from a combined expression.
func parseCombinedSpecs(text []byte, attrIdx, dataIdx int) (attrInner, dataInner []byte, err error) {
	extractInner := func(start int, prefix []byte) ([]byte, error) {
		inner, ok := bytes.CutPrefix(text[start:], prefix)
		if !ok {
			return nil, fmt.Errorf("expected %q at position %d", prefix, start)
		}

		closeIdx := bytes.IndexByte(inner, '}')
		if closeIdx == -1 {
			return nil, fmt.Errorf(
				"invalid format: expected '}' at end of value specifier")
		}

		return inner[:closeIdx], nil
	}

	attrInner, err = extractInner(attrIdx, attrPrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("attribute spec: %w", err)
	}

	dataInner, err = extractInner(dataIdx, dataPrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("data spec: %w", err)
	}

	return attrInner, dataInner, nil
}

// parseValues parses the value spec string, e.g.:
//
//	"date:date, tz=date_timezone"
func parseValues(s []byte) ([]ValueSpec, error) {
	s = bytes.TrimSpace(s)

	if len(s) == 0 {
		return nil, errors.New("no values were specified")
	}

	// Normalize commas to spaces so that both "a, b" and "a b" work.
	s = bytes.ReplaceAll(s, bComma, []byte(" "))
	parts := bytes.Fields(s)
	values := make([]ValueSpec, 0, len(parts))

	for _, part := range parts {
		var spec ValueSpec

		part, optional := bytes.CutSuffix(part, bQMark)

		spec.Optional = optional

		role, remainder, hasRole := bytes.Cut(part, bEqual)
		if hasRole {
			spec.Role = string(bytes.TrimSpace(role))
		} else {
			remainder = role
		}

		name, annotation, hasAnnotation := bytes.Cut(remainder, bColon)
		if hasAnnotation {
			spec.Annotation = string(bytes.TrimSpace(annotation))
		}

		spec.Name = string(name)

		values = append(values, spec)
	}

	return values, nil
}
