package test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type GoldenHelper interface {
	CmpOpts() cmp.Options
	JSONTransform(value map[string]any) error
}

type GoldenHelperForAny interface {
	JSONTransformAny(value any) error
}

var _ GoldenHelper = SortMapStringKeys{}

type SortMapStringKeys struct{}

// CmpOpts implements GoldenHelper.
func (fi SortMapStringKeys) CmpOpts() cmp.Options {
	return cmp.Options{
		cmpopts.SortMaps(strings.Compare),
	}
}

// JSONTransform implements GoldenHelper.
func (fi SortMapStringKeys) JSONTransform(value map[string]any) error {
	return nil
}

// TestAgainstGolden compares a result against the contents of the file at the
// goldenPath. Run with regenerate set to true to create or update the file.
func TestAgainstGolden[T any](
	t TestingT,
	regenerate bool,
	got T,
	goldenPath string,
	helpers ...GoldenHelper,
) {
	t.Helper()

	if regenerate {
		data, err := json.Marshal(got)
		Must(t, err, "marshal result")

		var (
			obj    any
			objMap map[string]any
		)

		switch reflect.TypeOf(got).Kind() {
		case reflect.Array:
			obj = []any{}
		case reflect.Map:
		case reflect.Struct:
			objMap = map[string]any{}
			obj = objMap
		default:
			var z T

			obj = z
		}

		err = json.Unmarshal(data, &obj)
		Must(t, err, "unmarshal for transform")

		for i := range helpers {
			anyHelper, hasAnyHelper := helpers[i].(GoldenHelperForAny)

			switch {
			case objMap != nil:
				err := helpers[i].JSONTransform(objMap)
				Must(t, err, "transform for storage")
			case hasAnyHelper:
				err := anyHelper.JSONTransformAny(obj)
				Must(t, err, "transform for storage")
			}
		}

		data, err = json.MarshalIndent(obj, "", "  ")
		Must(t, err, "marshal for storage in %q", goldenPath)

		// End all files with a newline
		data = append(data, '\n')

		err = os.WriteFile(goldenPath, data, 0o600)
		Must(t, err, "write golden file %q", goldenPath)
	}

	wantData, err := os.ReadFile(goldenPath)
	Must(t, err, "read from golden file %q", goldenPath)

	var wantValue T

	err = json.Unmarshal(wantData, &wantValue)
	Must(t, err, "unmarshal data from golden file %q", goldenPath)

	var cmpOpts cmp.Options

	for _, h := range helpers {
		cmpOpts = append(cmpOpts, h.CmpOpts()...)
	}

	EqualDiffWithOptions(t, wantValue, got, cmpOpts,
		"must match golden file %q", goldenPath)
}

// EqualMessage runs a cmp.Diff with protobuf-specific options.
func EqualDiffWithOptions[T any](
	t TestingT,
	want T, got T,
	opts cmp.Options,
	format string, a ...any,
) {
	t.Helper()

	diff := cmp.Diff(want, got, opts...)
	if diff != "" {
		msg := fmt.Sprintf(format, a...)
		t.Fatalf("%s: mismatch (-want +got):\n%s", msg, diff)
	}

	if testing.Verbose() {
		t.Logf("success: "+format, a...)
	}
}
