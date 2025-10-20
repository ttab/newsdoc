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
		"annotated":  ".meta(type='core/collection').links(rel='item').data{date:date, tz=date_timezone?}",
		"attributes": ".content(type='example/assumed-static-tz')@{value:date}",
		"multisel":   ".links(rel='point-in-time' type='example/pit').data{timestamp}",
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
				".meta(type='example/collection').links(rel='item').data{date:date, tz=date_timezone?}",
				".meta(type='example/collection').links(rel='item').data{start, end}",
				".content(type='example/assumed-static-tz')@{value:date}",
				".links(rel='point-in-time' type='example/pit').data{timestamp}",
			},
			Document: "constructed.json",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var doc newsdoc.Document

			err := test.UnmarshalFile(
				filepath.Join(dataDir, "constructed.json"),
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
