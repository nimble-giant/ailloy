package commands

import (
	"testing"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestVerifyManifestAgainstLock_NoLock(t *testing.T) {
	entries := []foundry.InstalledEntry{
		{Name: "a", Source: "github.com/x/a", Commit: "abc"},
	}
	if got := verifyManifestAgainstLock(entries, nil); got != nil {
		t.Errorf("expected nil failures with no lock, got %v", got)
	}
}

func TestVerifyManifestAgainstLock_CommitDrift(t *testing.T) {
	entries := []foundry.InstalledEntry{
		{Name: "a", Source: "github.com/x/a", Commit: "aaaaaaa1"},
	}
	lock := &foundry.LockFile{
		APIVersion: "v1",
		Molds: []foundry.LockEntry{
			{Name: "a", Source: "github.com/x/a", Version: "v1.0.0", Commit: "bbbbbbb2", Timestamp: time.Now()},
		},
	}
	failures := verifyManifestAgainstLock(entries, lock)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
}

func TestVerifyManifestAgainstLock_MissingFromLock(t *testing.T) {
	entries := []foundry.InstalledEntry{
		{Name: "a", Source: "github.com/x/a", Commit: "abc"},
		{Name: "b", Source: "github.com/x/b", Commit: "def"},
	}
	lock := &foundry.LockFile{
		APIVersion: "v1",
		Molds: []foundry.LockEntry{
			{Name: "a", Source: "github.com/x/a", Version: "v1.0.0", Commit: "abc", Timestamp: time.Now()},
		},
	}
	failures := verifyManifestAgainstLock(entries, lock)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing b, got %d: %v", len(failures), failures)
	}
}

func TestVerifyManifestAgainstLock_AllPresent(t *testing.T) {
	entries := []foundry.InstalledEntry{
		{Name: "a", Source: "github.com/x/a", Commit: "abc"},
	}
	lock := &foundry.LockFile{
		APIVersion: "v1",
		Molds: []foundry.LockEntry{
			{Name: "a", Source: "github.com/x/a", Version: "v1.0.0", Commit: "abc", Timestamp: time.Now()},
		},
	}
	if got := verifyManifestAgainstLock(entries, lock); got != nil {
		t.Errorf("expected no failures, got %v", got)
	}
}
