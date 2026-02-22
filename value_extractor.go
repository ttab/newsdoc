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
		return nil, fmt.Errorf("the expression cannot have both a .data{} and @{} value specifier")
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

	var accessor func(b Block, name string) string

	switch ve.ValueKind {
	case ValueKindAttributes:
		accessor = getBlockAttribute
	case ValueKindData:
		accessor = getBlockData
	case ValueKindBlock:
		panic("ValueKindBlock should have been handled above")
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
	Optional   bool   `json:",omitempty"`
	Annotation string `json:",omitempty"`
	Role       string `json:",omitempty"`
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

type ValueKind string

const (
	ValueKindAttributes ValueKind = "attributes"
	ValueKindData       ValueKind = "data"
	ValueKindBlock      ValueKind = "block"
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

// BlockSelector selects blocks by kind and optional attribute/data filters.
type BlockSelector struct {
	Kind BlockKind

	ID          *string      `json:",omitempty"`
	UUID        *string      `json:",omitempty"`
	URI         *string      `json:",omitempty"`
	URL         *string      `json:",omitempty"`
	Type        *string      `json:",omitempty"`
	Rel         *string      `json:",omitempty"`
	Role        *string      `json:",omitempty"`
	Name        *string      `json:",omitempty"`
	Value       *string      `json:",omitempty"`
	Contenttype *string      `json:",omitempty"`
	Sensitivity *string      `json:",omitempty"`
	DataFilters []DataFilter `json:",omitempty"`
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

func (bs BlockSelector) Filter(blocks []Block) []Block {
	return slices.Collect(bs.Iterator(slices.Values(blocks)))
}

func (bs BlockSelector) Matches(b Block) bool {
	switch {
	case bs.ID != nil && b.ID != *bs.ID:
		return false
	case bs.UUID != nil && b.UUID != *bs.UUID:
		return false
	case bs.URI != nil && b.URI != *bs.URI:
		return false
	case bs.URL != nil && b.URL != *bs.URL:
		return false
	case bs.Type != nil && b.Type != *bs.Type:
		return false
	case bs.Rel != nil && b.Rel != *bs.Rel:
		return false
	case bs.Role != nil && b.Role != *bs.Role:
		return false
	case bs.Name != nil && b.Name != *bs.Name:
		return false
	case bs.Value != nil && b.Value != *bs.Value:
		return false
	case bs.Contenttype != nil && b.Contenttype != *bs.Contenttype:
		return false
	case bs.Sensitivity != nil && b.Sensitivity != *bs.Sensitivity:
		return false
	}

	for _, df := range bs.DataFilters {
		switch df.Mode {
		case DataFilterExact:
			if b.Data.Get(df.Key, "") != df.Value {
				return false
			}
		case DataFilterExists:
			_, ok := b.Data[df.Key]
			if !ok {
				return false
			}
		case DataFilterNonEmpty:
			if b.Data.Get(df.Key, "") == "" {
				return false
			}
		}
	}

	return true
}

func strPtr(s string) *string {
	return &s
}

// setSelectorField assigns a value to the correct field in BlockSelector based
// on the key.
func setSelectorField(selector *BlockSelector, key, value string) error {
	vPtr := strPtr(value)

	switch key {
	case "id":
		selector.ID = vPtr
	case "uuid":
		selector.UUID = vPtr
	case "uri":
		selector.URI = vPtr
	case "url":
		selector.URL = vPtr
	case "type":
		selector.Type = vPtr
	case "rel":
		selector.Rel = vPtr
	case "role":
		selector.Role = vPtr
	case "name":
		selector.Name = vPtr
	case "value":
		selector.Value = vPtr
	case "contenttype":
		selector.Contenttype = vPtr
	case "sensitivity":
		selector.Sensitivity = vPtr
	default:
		return fmt.Errorf("unknown attribute key: %s", key)
	}

	return nil
}

// parseAttributes parses the attribute string, e.g.:
//
//	"type='core/text' rel='item'"
//	"type='core/event' data.date?? data.status='confirmed'"
func parseAttributes(selector *BlockSelector, attrsStr []byte) error {
	remaining := bytes.TrimSpace(attrsStr)

	for len(remaining) != 0 {
		// Check for data filter prefix before trying to parse as
		// a regular attribute.
		if bytes.HasPrefix(remaining, bDataDot) {
			df, n, err := parseDataFilter(remaining)
			if err != nil {
				return err
			}

			selector.DataFilters = append(selector.DataFilters, df)
			remaining = bytes.TrimSpace(remaining[n:])

			continue
		}

		key, valPart, foundEq := bytes.Cut(remaining, bEqual)
		if !foundEq {
			return fmt.Errorf("invalid attribute format, expected '=' in: %q", remaining)
		}

		key = bytes.TrimSpace(key)

		valPart = bytes.TrimSpace(valPart)
		if len(valPart) == 0 {
			return fmt.Errorf("missing value for attribute key: %q", key)
		}

		quote := valPart[0]
		if quote != '\'' {
			return fmt.Errorf("attribute value must be quoted: %q", valPart)
		}

		// Find the closing quote, starting after the opening one,
		// skipping escaped quotes (\').
		endQuoteIndex := findClosingQuote(valPart[1:])
		if endQuoteIndex == -1 {
			return fmt.Errorf("unterminated quoted value in: %q", valPart)
		}

		// The index is relative to valPart[1:], increment to get the
		// absolute index in valPart.
		endQuoteIndex++

		value := unescapeQuoted(valPart[1:endQuoteIndex])

		// Assign the value to the selector.
		if err := setSelectorField(selector, string(key), string(value)); err != nil {
			return err
		}

		remaining = bytes.TrimSpace(valPart[endQuoteIndex+1:])
	}

	return nil
}

// parseDataFilter parses a data filter token from the start of b. It returns
// the parsed filter and the number of bytes consumed. The input must start with
// "data.".
func parseDataFilter(b []byte) (DataFilter, int, error) {
	// Find the end of this token (next space or end of input).
	tokenEnd := bytes.IndexByte(b, ' ')
	if tokenEnd == -1 {
		tokenEnd = len(b)
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
			attrsStr = attrsStr[:len(attrsStr)-1]

			if err := parseAttributes(&selector, attrsStr); err != nil {
				return nil, err
			}
		}

		selectors = append(selectors, selector)
	}

	return selectors, nil
}

// parseValues parses the value spec string, e.g.:
//
//	"date:date, tz=date_timezone"
func parseValues(s []byte) ([]ValueSpec, error) {
	s = bytes.TrimSpace(s)

	if len(s) == 0 {
		return nil, errors.New("no values were specified")
	}

	parts := bytes.Split(s, bComma)
	values := make([]ValueSpec, 0, len(parts))

	for _, part := range parts {
		part = bytes.TrimSpace(part)
		if len(part) == 0 {
			continue
		}

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
