package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
)

type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
}

func Mustf(t TestingT, err error, format string, a ...any) {
	t.Helper()

	if err != nil {
		t.Fatalf("failed: %s: %v", fmt.Sprintf(format, a...), err)
	}

	if testing.Verbose() {
		t.Logf("success: "+format, a...)
	}
}

func Regenerate() bool {
	return os.Getenv("REGENERATE") == "true"
}

// UnmarshalFile is a utility function for reading and unmarshalling a file
// containing JSON. The parsing will be strict and disallow unknown fields.
func UnmarshalFile(path string, o any) (outErr error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			outErr = errors.Join(outErr, fmt.Errorf(
				"failed to close file: %w", err))
		}
	}()

	dec := json.NewDecoder(f)

	dec.DisallowUnknownFields()

	err = dec.Decode(o)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}
