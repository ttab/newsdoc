package newsdoc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ttab/newsdoc"
	"github.com/ttab/newsdoc/internal/test"
)

const coreText = "core/text"

func TestValueExtractorParse(t *testing.T) {
	regenerate := test.Regenerate()
	dataDir := filepath.Join("testdata", t.Name())

	err := os.MkdirAll(dataDir, 0o770)
	test.Mustf(t, err, "ensure testdata dir")

	cases := map[string]string{
		"annotated":        ".meta(type='core/collection').links(rel='item').data{date:date, tz=date_timezone?}",
		"attributes":       ".content(type='example/assumed-static-tz')@{value:date}",
		"multisel":         ".links(rel='point-in-time' type='example/pit').data{timestamp}",
		"block":            "items=.meta(type='example/collection').links(rel='item'):thing",
		"block_no_annot":   "items=.meta(type='example/collection').links(rel='item')",
		"data_exact":       ".meta(type='core/event' data.status='confirmed').data{date}",
		"data_exists":      ".meta(type='core/event' data.date?).data{date}",
		"data_non_empty":   ".meta(type='core/event' data.date??).data{date}",
		"data_multi_mixed": ".meta(type='core/event' data.date?? data.status='confirmed').data{date}",
		"child_selector":   "assignment=.meta(type='core/assignment')#.links(rel='deliverable' uuid='abc'):label",

		// Special characters and sequences inside quoted strings.
		"quoted_hash_in_type":          ".meta(type='core/thing#v1').data{date}",
		"quoted_hash_in_block":         "items=.meta(type='core/tagged#v2').links(rel='item')",
		"quoted_at_in_rel":             ".meta(type='core/section').links(rel='item@2').data{date}",
		"quoted_at_brace_in_value":     ".content(value='@{placeholder}')@{title}",
		"quoted_at_brace_in_type":      ".meta(type='x@{y}').data{date}",
		"quoted_dotdata_in_type":       ".meta(type='test.data.value').data{date}",
		"quoted_dotdata_brace_type":    ".meta(type='.data{tricky}').data{date}",
		"quoted_dotdata_brace_filt":    ".meta(type='core/event' data.format='.data{x}').data{date}",
		"quoted_hash_in_data_filter":   ".meta(type='core/event' data.tag='news#breaking').data{date}",
		"escaped_quote_in_data_filter": ".meta(type='core/event' data.tag='it\\'s breaking').data{date}",

		// OR operator and parenthesized grouping.
		"or_simple":      ".meta(value='text' or value='picture').data{date}",
		"or_with_type":   ".meta(type='core/thing' (value='a' or value='b')).data{date}",
		"or_three_way":   ".meta(value='text' or value='picture' or value='video').data{date}",
		"grouped_or_and": ".meta((type='a' value='x') or (type='b' value='y')).data{date}",
		"or_data_filter": ".meta(data.status='draft' or data.status='review').data{date}",
		"nested_groups":  ".meta((type='a' (value='x' or value='y')) or (type='b' value='z')).data{date}",
		"or_data_exists": ".meta(type='core/event' (data.date?? or data.start??)).data{date}",

		// Combined attribute and data extraction.
		"combined_attr_data": ".meta(type='core/assignment')@{title}.data{start_date date_tz}",
	}

	for name, str := range cases {
		t.Run(name, func(t *testing.T) {
			ve, err := newsdoc.ValueExtractorFromString(str)
			test.Mustf(t, err, "parse expression %q", str)

			test.AgainstGolden(t, regenerate, ve,
				filepath.Join(dataDir, name+".json"))
		})
	}
}

func TestValueExtractorParseErrors(t *testing.T) {
	cases := map[string]string{
		"unmatched_open_paren":  ".meta((type='a').data{date}",
		"unmatched_close_paren": ".meta(type='a')).data{date}",
		"empty_group":           ".meta(()).data{date}",
		"leading_or":            ".meta(or type='a').data{date}",
		"trailing_or":           ".meta(type='a' or).data{date}",
		"double_or":             ".meta(type='a' or or type='b').data{date}",

		// Additional error cases from TS port.
		"block_no_selectors":          "block=",
		"empty_block_name":            "=.meta(type='a')",
		"data_on_document":            ".data{date}",
		"missing_close_brace_data":    ".meta(type='a').data{date",
		"missing_close_brace_attr":    ".meta(type='a')@{title",
		"no_name_or_value":            "just-text-without-equals-or-braces",
		"combined_no_selector":        "@{title}.data{date}",
		"combined_missing_close_data": ".meta(type='a')@{title}.data{date",
		"unknown_block_kind":          ".widgets(type='a').data{date}",
		"empty_values":                ".meta(type='a').data{}",
		"mismatched_paren":            "block=.meta(type='a'",
		"double_dot":                  ".meta..links.data{date}",
		"invalid_data_filter":         ".meta(data.key).data{date}",
		"unterminated_data_value":     ".meta(data.key='unterminated).data{date}",
		"unknown_attr_key":            ".meta(foo='bar').data{date}",
		"unquoted_attr_value":         ".meta(type=bar).data{date}",
		"missing_attr_value":          ".meta(type=).data{date}",
		"unterminated_attr_value":     ".meta(type='bar).data{date}",
		"unquoted_data_value":         ".meta(data.key=bar).data{date}",
		"empty_data_key_exists":       ".meta(data.?).data{date}",
		"empty_data_key_non_empty":    ".meta(data.??).data{date}",
		"empty_data_key_exact":        ".meta(data.='val').data{date}",
		"empty_after_open_paren":      ".meta(type='a' ()).data{date}",
		"attr_no_equals":              ".meta(type='a' noequals).data{date}",
		"open_paren_eof":              "block=.meta(type='a' ()",
		"child_selector_no_dot":       "block=.meta(type='a')#links(rel='b')",
		"combined_empty_attr_values":  ".meta(type='a')@{}.data{date}",
		"combined_empty_data_values":  ".meta(type='a')@{title}.data{}",
		"combined_missing_close_attr": ".meta(type='a')@{title.data{date",
	}

	for name, str := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newsdoc.ValueExtractorFromString(str)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", str)
			}

			t.Logf("got expected error: %v", err)
		})
	}
}

func TestBlockSelectorFilterBlocks(t *testing.T) {
	blocks := []newsdoc.Block{
		{Type: coreText, Value: "hello"},
		{Type: "core/image", Title: "photo"},
		{Type: coreText, Value: "world"},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/text').data{val}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	sel := ve.Selectors[0]
	filtered := sel.FilterBlocks(blocks)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered blocks, got %d", len(filtered))
	}

	for _, b := range filtered {
		if b.Type != coreText {
			t.Errorf("unexpected type %q in filtered results", b.Type)
		}
	}
}

func TestBlockSelectorFilterBlocksNoFilter(t *testing.T) {
	blocks := []newsdoc.Block{
		{Type: "core/text"},
		{Type: "core/image"},
	}

	sel := newsdoc.BlockSelector{Kind: newsdoc.BlockKindMeta}
	filtered := sel.FilterBlocks(blocks)

	if len(filtered) != 2 {
		t.Fatalf("expected all blocks when no filter, got %d", len(filtered))
	}
}

func TestBlockSelectorMatches(t *testing.T) {
	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/text').data{val}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	sel := ve.Selectors[0]

	if !sel.Matches(newsdoc.Block{Type: coreText}) {
		t.Error("should match block with matching type")
	}

	if sel.Matches(newsdoc.Block{Type: "core/image"}) {
		t.Error("should not match block with different type")
	}
}

func TestBlockSelectorMatchesNoFilter(t *testing.T) {
	sel := newsdoc.BlockSelector{Kind: newsdoc.BlockKindMeta}

	// Nil filter should match everything.
	if !sel.Matches(newsdoc.Block{Type: "anything"}) {
		t.Error("nil filter should match any block")
	}
}

func TestCollectDocumentAttributes(t *testing.T) {
	doc := newsdoc.Document{
		UUID:     "test-uuid",
		Type:     "core/article",
		Title:    "My Title",
		Language: "en",
		URI:      "article://test",
		URL:      "https://example.com",
	}

	ve, err := newsdoc.ValueExtractorFromString("@{title}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["title"].Value != "My Title" {
		t.Errorf("expected title 'My Title', got %q", results[0]["title"].Value)
	}
}

func TestCollectDocumentAttributeMissing(t *testing.T) {
	doc := newsdoc.Document{}

	ve, err := newsdoc.ValueExtractorFromString("@{uuid}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 0 {
		t.Errorf("expected no results for missing required attribute, got %d", len(results))
	}
}

func TestCollectDocumentAttributeOptional(t *testing.T) {
	doc := newsdoc.Document{Title: "Has Title"}

	ve, err := newsdoc.ValueExtractorFromString("@{title, uuid?}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result with optional missing, got %d", len(results))
	}

	if _, ok := results[0]["uuid"]; ok {
		t.Error("optional missing uuid should not be in results")
	}

	if results[0]["title"].Value != "Has Title" {
		t.Errorf("title should be extracted, got %q", results[0]["title"].Value)
	}
}

func TestCollectNoMatchingBlocks(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{Type: "core/newsvalue"},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/nonexistent').data{date}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 0 {
		t.Errorf("expected empty results for non-matching selector, got %d", len(results))
	}
}

func TestCollectFilterlessSelector(t *testing.T) {
	doc := newsdoc.Document{
		Links: []newsdoc.Block{
			{Type: "core/author", Title: "Jane"},
			{Type: "core/subject", Title: "News"},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(".links@{title}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 2 {
		t.Fatalf("expected 2 results from filterless selector, got %d", len(results))
	}
}

func TestCollectOrFilter(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{Type: coreText, Value: "hello", Data: newsdoc.DataMap{"date": "2024-01-01"}},
			{Type: "core/image", Value: "pic", Data: newsdoc.DataMap{"date": "2024-01-02"}},
			{Type: "core/video", Value: "vid", Data: newsdoc.DataMap{"date": "2024-01-03"}},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/text' or type='core/video').data{date}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 2 {
		t.Fatalf("expected 2 results from OR filter, got %d", len(results))
	}
}

func TestCollectContentBlocks(t *testing.T) {
	doc := newsdoc.Document{
		Content: []newsdoc.Block{
			{Type: coreText, Value: "Hello World"},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".content(type='core/text')@{value}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["value"].Value != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", results[0]["value"].Value)
	}
}

func TestCollectLinksBlocks(t *testing.T) {
	doc := newsdoc.Document{
		Links: []newsdoc.Block{
			{Type: "core/author", UUID: "author-uuid"},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".links(type='core/author')@{uuid}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["uuid"].Value != "author-uuid" {
		t.Errorf("expected 'author-uuid', got %q", results[0]["uuid"].Value)
	}
}

func TestCollectAllDocumentAttributes(t *testing.T) {
	doc := newsdoc.Document{
		UUID:     "u",
		Type:     "t",
		URI:      "uri",
		URL:      "url",
		Title:    "title",
		Language: "lang",
	}

	attrs := []string{"uuid", "type", "uri", "url", "title", "language"}
	for _, attr := range attrs {
		ve, err := newsdoc.ValueExtractorFromString("@{" + attr + "}")
		if err != nil {
			t.Fatalf("parse @{%s}: %v", attr, err)
		}

		results := ve.Collect(doc)
		if len(results) == 0 {
			t.Errorf("@{%s} returned no results", attr)

			continue
		}

		if results[0][attr].Value == "" {
			t.Errorf("@{%s} returned empty value", attr)
		}
	}
}

func TestCollectAllBlockAttributes(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				ID:          "id-val",
				UUID:        "uuid-val",
				URI:         "uri-val",
				URL:         "url-val",
				Type:        "type-val",
				Title:       "title-val",
				Rel:         "rel-val",
				Role:        "role-val",
				Name:        "name-val",
				Value:       "value-val",
				Contenttype: "ct-val",
				Sensitivity: "sens-val",
			},
		},
	}

	attrs := []string{
		"id", "uuid", "uri", "url", "type", "title",
		"rel", "role", "name", "value", "contenttype", "sensitivity",
	}

	for _, attr := range attrs {
		ve, err := newsdoc.ValueExtractorFromString(".meta@{" + attr + "}")
		if err != nil {
			t.Fatalf("parse .meta@{%s}: %v", attr, err)
		}

		results := ve.Collect(doc)
		if len(results) == 0 {
			t.Errorf(".meta@{%s} returned no results", attr)

			continue
		}

		if results[0][attr].Value == "" {
			t.Errorf(".meta@{%s} returned empty value", attr)
		}
	}
}

func TestCollectUnknownDocAttribute(t *testing.T) {
	doc := newsdoc.Document{Title: "Test"}

	ve, err := newsdoc.ValueExtractorFromString("@{nonexistent}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 0 {
		t.Errorf("expected no results for unknown attribute, got %d", len(results))
	}
}

func TestCollectCombinedOptionalMissing(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type:  "core/assignment",
				Title: "My Assignment",
				Data:  newsdoc.DataMap{"start_date": "2024-01-01"},
			},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/assignment')@{title}.data{start_date date_tz?}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if _, ok := results[0]["date_tz"]; ok {
		t.Error("missing optional value should not be in results")
	}

	if results[0]["title"].Value != "My Assignment" {
		t.Errorf("title should be extracted, got %q", results[0]["title"].Value)
	}
}

func TestCollectCombinedRequiredMissing(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/assignment",
				Data: newsdoc.DataMap{"start_date": "2024-01-01"},
			},
		},
	}

	// title is required in @{} but the block has no title.
	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/assignment')@{title}.data{start_date}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 0 {
		t.Errorf("expected no results when required combined attr missing, got %d", len(results))
	}
}

func TestCollectAnnotationAndRole(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/collection",
				Links: []newsdoc.Block{
					{
						Rel:  "item",
						Data: newsdoc.DataMap{"date": "2024-06-15", "date_timezone": "Europe/Stockholm"},
					},
				},
			},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/collection').links(rel='item').data{date:date, tz=date_timezone?}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	dateVal := results[0]["date"]
	if dateVal.Annotation != "date" {
		t.Errorf("expected annotation 'date', got %q", dateVal.Annotation)
	}

	tzVal := results[0]["date_timezone"]
	if tzVal.Role != "tz" {
		t.Errorf("expected role 'tz', got %q", tzVal.Role)
	}
}

func TestCollectChildSelectorContent(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/section",
				Content: []newsdoc.Block{
					{Type: coreText, Value: "included"},
				},
			},
			{
				Type:    "core/section",
				Content: []newsdoc.Block{},
			},
		},
	}

	// Child selector using content kind.
	ve, err := newsdoc.ValueExtractorFromString(
		"section=.meta(type='core/section')#.content(type='core/text')")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only section with matching content child), got %d", len(results))
	}
}

func TestCollectChildSelectorMeta(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/assignment",
				Meta: []newsdoc.Block{
					{Type: "core/status", Value: "done"},
				},
			},
			{
				Type: "core/assignment",
				Meta: []newsdoc.Block{
					{Type: "core/status", Value: "pending"},
				},
			},
		},
	}

	// Child selector using meta kind with value filter.
	ve, err := newsdoc.ValueExtractorFromString(
		"done=.meta(type='core/assignment')#.meta(type='core/status' value='done')")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only assignment with done status), got %d", len(results))
	}
}

func TestCollectMultiLevelChildSelector(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/wrapper",
				Links: []newsdoc.Block{
					{
						Type: "core/ref",
						Rel:  "child",
						Content: []newsdoc.Block{
							{Type: "core/marker", Value: "deep"},
						},
					},
				},
			},
			{
				Type: "core/wrapper",
				Links: []newsdoc.Block{
					{
						Type: "core/ref",
						Rel:  "child",
					},
				},
			},
		},
	}

	// Multi-level child selector: wrapper must have links with content matching.
	ve, err := newsdoc.ValueExtractorFromString(
		"w=.meta(type='core/wrapper')#.links(rel='child').content(type='core/marker')")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only wrapper with deep marker), got %d", len(results))
	}
}

func TestCollectMultiLevelMainSelector(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/section",
				Content: []newsdoc.Block{
					{Type: coreText, Value: "inner", Data: newsdoc.DataMap{"key": "val"}},
				},
			},
		},
	}

	// Multi-level main selector chain going meta -> content.
	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/section').content(type='core/text').data{key}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["key"].Value != "val" {
		t.Errorf("expected 'val', got %q", results[0]["key"].Value)
	}
}

func TestCollectMultiLevelMainSelectorLinks(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/wrapper",
				Links: []newsdoc.Block{
					{Rel: "ref", UUID: "target-uuid"},
				},
			},
		},
	}

	// Main selector chain going meta -> links.
	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/wrapper').links(rel='ref')@{uuid}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["uuid"].Value != "target-uuid" {
		t.Errorf("expected 'target-uuid', got %q", results[0]["uuid"].Value)
	}
}

func TestCollectMultiLevelMainSelectorMeta(t *testing.T) {
	doc := newsdoc.Document{
		Content: []newsdoc.Block{
			{
				Type: "core/article",
				Meta: []newsdoc.Block{
					{Type: "core/flag", Value: "important"},
				},
			},
		},
	}

	// Main selector chain going content -> meta.
	ve, err := newsdoc.ValueExtractorFromString(
		".content(type='core/article').meta(type='core/flag')@{value}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["value"].Value != "important" {
		t.Errorf("expected 'important', got %q", results[0]["value"].Value)
	}
}

func TestCollectCombinedReversedOrder(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type:  "core/item",
				Title: "Item Title",
				Data:  newsdoc.DataMap{"date": "2024-01-01"},
			},
		},
	}

	// Combined with data before attributes (.data{} before @{}).
	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/item').data{date}@{title}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0]["date"].Value != "2024-01-01" {
		t.Errorf("expected date '2024-01-01', got %q", results[0]["date"].Value)
	}

	if results[0]["title"].Value != "Item Title" {
		t.Errorf("expected title 'Item Title', got %q", results[0]["title"].Value)
	}
}

func TestCollectChildSelectorInnerLinks(t *testing.T) {
	// Tests the links branch in hasMatchingChildren's inner loop (second+
	// child selector with Kind=links).
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/wrapper",
				Content: []newsdoc.Block{
					{
						Type: "core/ref",
						Links: []newsdoc.Block{
							{Rel: "target", UUID: "found"},
						},
					},
				},
			},
			{
				Type:    "core/wrapper",
				Content: []newsdoc.Block{},
			},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		"w=.meta(type='core/wrapper')#.content(type='core/ref').links(rel='target')")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestCollectChildSelectorInnerMeta(t *testing.T) {
	// Tests the meta branch in hasMatchingChildren's inner loop (second+
	// child selector with Kind=meta).
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{
				Type: "core/container",
				Links: []newsdoc.Block{
					{
						Rel: "item",
						Meta: []newsdoc.Block{
							{Type: "core/flag", Value: "important"},
						},
					},
				},
			},
			{
				Type: "core/container",
				Links: []newsdoc.Block{
					{Rel: "item"},
				},
			},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		"c=.meta(type='core/container')#.links(rel='item').meta(type='core/flag')")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestCollectDataFilterExists(t *testing.T) {
	doc := newsdoc.Document{
		Meta: []newsdoc.Block{
			{Type: "core/item", Data: newsdoc.DataMap{"status": "draft"}},
			{Type: "core/item", Data: newsdoc.DataMap{"other": "val"}},
			{Type: "core/item"},
		},
	}

	ve, err := newsdoc.ValueExtractorFromString(
		".meta(type='core/item' data.status?).data{status}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	results := ve.Collect(doc)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only block with status key), got %d", len(results))
	}
}

type extractorCase struct {
	Expressions []string
	Document    string
}

func TestValueExtractor(t *testing.T) {
	regenerate := test.Regenerate()
	dataDir := filepath.Join("testdata", t.Name())

	err := os.MkdirAll(dataDir, 0o770)
	test.Mustf(t, err, "ensure testdata dir")

	cases := map[string]extractorCase{
		"constructed": {
			Expressions: []string{
				"@{title}",
				".meta(type='example/collection').links(rel='item').data{date:date, tz=date_timezone?}",
				".meta(type='example/collection').links(rel='item').data{start, end}",
				".content(type='example/assumed-static-tz')@{value:date}",
				".links(rel='point-in-time' type='example/pit').data{timestamp}",
				"pointy=.links(rel='point-in-time' type='example/pit'):interesting",
				"unpointy=.links(rel='point-in-time' type='example/pit')",
				// Data filter: exact match on date_timezone.
				".meta(type='example/collection').links(rel='item' data.date_timezone='Asia/Shanghai').data{date:date}",
				// Data filter: key exists (date exists on 2 of 3 items).
				".meta(type='example/collection').links(rel='item' data.date?).data{date:date}",
				// Data filter: key non-empty (date non-empty on 2 of 3 items).
				".meta(type='example/collection').links(rel='item' data.date??).data{date:date}",
			},
			Document: "constructed.json",
		},
		"planning": {
			Expressions: []string{
				".meta(type='core/planning-item').data{start_date, date_tz?}",
				".meta(type='core/assignment').links(rel='deliverable')@{uuid}",
				"block=.meta(type='core/assignment').links(rel='deliverable' data.nonesuch='value')",
				// Child selector: get all assignment that reference a given deliverable.
				"assignment=.meta(type='core/assignment')#.links(rel='deliverable' uuid='4f13347f-04b3-4f22-a992-9316d824b81f')",
				// Child selector with value.
				".meta(type='core/assignment')@{id}#.links(rel='deliverable' uuid='4f13347f-04b3-4f22-a992-9316d824b81f')",
				// Extract both data and attributes.
				".meta(type='core/assignment')@{title}.data{start_date date_tz}",
			},
			Document: "planning.json",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var doc newsdoc.Document

			err := test.UnmarshalFile(
				filepath.Join(dataDir, c.Document),
				&doc)
			test.Mustf(t, err, "unmarshal document")

			var extracted [][]newsdoc.ExtractedItems

			for _, exp := range c.Expressions {
				ve, err := newsdoc.ValueExtractorFromString(exp)
				test.Mustf(t, err, "parse expression %q", exp)

				extracted = append(extracted, ve.Collect(doc))
			}

			test.AgainstGolden(t, regenerate, extracted,
				filepath.Join(dataDir, name+"_extracted.json"),
				test.SortMapStringKeys{},
			)
		})
	}
}
