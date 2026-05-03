package merge

import (
	"errors"
	"testing"
)

func TestParseError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("oops")
	pe := &ParseError{Path: "x.json", Format: "json", Err: inner}
	if got := pe.Error(); got == "" {
		t.Fatal("ParseError.Error returned empty string")
	}
	if !errors.Is(pe, inner) {
		t.Fatal("errors.Is should match wrapped inner")
	}
}
