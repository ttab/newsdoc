package newsdoc

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
)

type ValueExtractor struct {
	Selectors []BlockSelector
	ValueKind ValueKind
	Values    []ValueSpec
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
)

func ValueExtractorFromString(text string) (*ValueExtractor, error) {
	return ValueExtractorFromBytes([]byte(text))
}

func ValueExtractorFromBytes(text []byte) (*ValueExtractor, error) {
	var ve ValueExtractor

	// Find the split point between selectors and value spec
	dataIdx := bytes.LastIndex(text, dataPrefix)
	attrIdx := bytes.LastIndex(text, attrPrefix)

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
		return nil, fmt.Errorf("invalid format: missing .data{} or @{} value specifier")
	}

	if !bytes.HasSuffix(valueSpecInner, valuesEnd) {
		return nil, fmt.Errorf("invalid format: expected '}' at end of value specifier")
	}

	valueSpecInner = valueSpecInner[:len(valueSpecInner)-1]

	selectors, err := parseSelectors(selector)
	if err != nil {
		return nil, err
	}

	ve.Selectors = selectors

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

	var extracts []ExtractedItems

	var accessor func(b Block, name string) string

	switch ve.ValueKind {
	case ValueKindAttributes:
		accessor = getBlockAttribute
	case ValueKindData:
		accessor = getBlockData
	default:
		panic(fmt.Sprintf("unexpected newsdoc.ValueKind: %#v", ve.ValueKind))
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

func getDocumentAttribute(doc Document, name string) string {
	switch name {
	case "uuid":
		return doc.UUID
	case "type":
		return doc.Type
	case "uri":
		return doc.URI
	case "url":
		return doc.URL
	case "title":
		return doc.Title
	case "language":
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

func getBlockAttribute(b Block, name string) string {
	switch strings.ToLower(name) {
	case "id":
		return b.ID
	case "uuid":
		return b.UUID
	case "uri":
		return b.URI
	case "url":
		return b.URL
	case "type":
		return b.Type
	case "title":
		return b.Title
	case "rel":
		return b.Rel
	case "role":
		return b.Role
	case "name":
		return b.Name
	case "value":
		return b.Value
	case "contenttype":
		return b.Contenttype
	case "sensitivity":
		return b.Sensitivity
	default:
		return ""
	}
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
	Value      string
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
)

type BlockSelector struct {
	Kind BlockKind

	ID          *string `json:",omitempty"`
	URI         *string `json:",omitempty"`
	URL         *string `json:",omitempty"`
	Type        *string `json:",omitempty"`
	Rel         *string `json:",omitempty"`
	Role        *string `json:",omitempty"`
	Name        *string `json:",omitempty"`
	Value       *string `json:",omitempty"`
	Contenttype *string `json:",omitempty"`
	Sensitivity *string `json:",omitempty"`
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

func (bs BlockSelector) Filter(blocks []Block) []Block {
	return slices.Collect(bs.Iterator(slices.Values(blocks)))
}

func (bs BlockSelector) Matches(b Block) bool {
	switch {
	case bs.ID != nil && b.ID != *bs.ID:
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
func parseAttributes(selector *BlockSelector, attrsStr []byte) error {
	remaining := bytes.TrimSpace(attrsStr)
	for len(remaining) != 0 {
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

		// Find the closing quote, starting after the opening one
		endQuoteIndex := bytes.IndexByte(valPart[1:], quote)
		if endQuoteIndex == -1 {
			return fmt.Errorf("unterminated quoted value in: %q", valPart)
		}

		// The index is relative to valPart[1:], increment to get the
		// absolute index in valPart
		endQuoteIndex++

		value := valPart[1:endQuoteIndex]

		// Assign the value to the selector
		if err := setSelectorField(selector, string(key), string(value)); err != nil {
			return err
		}

		remaining = bytes.TrimSpace(valPart[endQuoteIndex+1:])
	}

	return nil
}

// parseSelectors manually parses the selector chain, e.g.:
//
//	".meta(type='example/thing').links"
func parseSelectors(s []byte) ([]BlockSelector, error) {
	if len(s) == 0 {
		return nil, nil
	}

	if !bytes.HasPrefix(s, bPeriod) {
		return nil, fmt.Errorf("selector chain must start with '.'")
	}

	parts := bytes.Split(s[1:], bPeriod)
	selectors := make([]BlockSelector, 0, len(parts))

	for _, part := range parts {
		if len(part) == 0 {
			return nil, fmt.Errorf("empty selector part found (double dot '..')")
		}

		kindStr, attrsStr, foundParen := bytes.Cut(part, bStartParen)

		var selector BlockSelector

		// Set and validate BlockKind
		switch BlockKind(kindStr) {
		case BlockKindMeta, BlockKindLinks, BlockKindContent:
			selector.Kind = BlockKind(kindStr)
		default:
			return nil, fmt.Errorf("unknown block kind: %s", kindStr)
		}

		// If there are parentheses, parse the attributes inside them
		if foundParen {
			if !bytes.HasSuffix(attrsStr, bEndParen) {
				return nil, fmt.Errorf("mismatched parenthesis in selector: %q", part)
			}

			// Remove the trailing ')' before parsing
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
