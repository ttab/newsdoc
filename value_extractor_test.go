package newsdoc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ttab/newsdoc"
	"github.com/ttab/newsdoc/internal/test"
)

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
				"assignment=.meta(type='core/assignment')#.links(rel='deliverable' uuid='4f13347f-04b3-4f22-a992-9316d824b81f')",
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
