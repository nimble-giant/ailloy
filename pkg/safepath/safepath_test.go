package safepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClean_RelativePath(t *testing.T) {
	result, err := Clean("foo/bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, "foo", "bar")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestClean_AbsolutePath(t *testing.T) {
	result, err := Clean("/tmp/test/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/test/path" {
		t.Errorf("expected /tmp/test/path, got %s", result)
	}
}

func TestClean_RemovesDotDot(t *testing.T) {
	result, err := Clean("/tmp/test/../other")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/other" {
		t.Errorf("expected /tmp/other, got %s", result)
	}
}

func TestClean_RemovesDot(t *testing.T) {
	result, err := Clean("/tmp/./test/./path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/test/path" {
		t.Errorf("expected /tmp/test/path, got %s", result)
	}
}

func TestValidateUnder_ValidPath(t *testing.T) {
	result, err := ValidateUnder("/tmp", "/tmp/subdir/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/subdir/file.txt" {
		t.Errorf("expected /tmp/subdir/file.txt, got %s", result)
	}
}

func TestValidateUnder_ExactBase(t *testing.T) {
	result, err := ValidateUnder("/tmp", "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp" {
		t.Errorf("expected /tmp, got %s", result)
	}
}

func TestValidateUnder_TraversalAttack(t *testing.T) {
	_, err := ValidateUnder("/tmp/safe", "/tmp/safe/../../etc/passwd")
	if err == nil {
		t.Error("expected error for traversal attack, got nil")
	}
}

func TestValidateUnder_OutsideBase(t *testing.T) {
	_, err := ValidateUnder("/tmp/safe", "/etc/passwd")
	if err == nil {
		t.Error("expected error for path outside base, got nil")
	}
}

func TestValidateUnder_SimilarPrefix(t *testing.T) {
	// /tmp/safe-extra should NOT be considered under /tmp/safe
	_, err := ValidateUnder("/tmp/safe", "/tmp/safe-extra/file.txt")
	if err == nil {
		t.Error("expected error for similar prefix path, got nil")
	}
}

func TestJoin_ValidJoin(t *testing.T) {
	result, err := Join("/tmp/base", "subdir", "file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/base/subdir/file.txt" {
		t.Errorf("expected /tmp/base/subdir/file.txt, got %s", result)
	}
}

func TestJoin_TraversalAttempt(t *testing.T) {
	_, err := Join("/tmp/base", "..", "..", "etc", "passwd")
	if err == nil {
		t.Error("expected error for traversal via join, got nil")
	}
}

func TestJoin_SingleElement(t *testing.T) {
	result, err := Join("/tmp/base", "file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/base/file.txt" {
		t.Errorf("expected /tmp/base/file.txt, got %s", result)
	}
}

func TestIsTraversal_WithDotDot(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"../secret", true},
		{"../../etc/passwd", true},
		{"foo/../bar", true},
		{"foo/bar/../../baz", true},
		{"safe/path", false},
		{"file.txt", false},
		{"/absolute/path", false},
		{"..hidden", true}, // contains ".." substring
		{"a..b", true},     // contains ".." substring
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsTraversal(tt.path)
			if result != tt.expected {
				t.Errorf("IsTraversal(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsTraversal_EmptyPath(t *testing.T) {
	result := IsTraversal("")
	if result {
		t.Error("expected empty path to not be a traversal")
	}
}
