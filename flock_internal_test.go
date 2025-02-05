package gflock

import (
	"os"
	"testing"
)

func Test(t *testing.T) {
	tmpFileFh, err := os.CreateTemp(os.TempDir(), "gflock-")
	if err != nil {
		t.Fatal(err)
	}
	tmpFileFh.Close()
	tmpFile := tmpFileFh.Name()
	os.Remove(tmpFile)

	lock := New(tmpFile)
	locked, err := lock.TryLock()
	if locked == false || err != nil {
		t.Fatalf("failed to lock: locked: %t, err: %v", locked, err)
	}

	newLock := New(tmpFile)
	locked, err = newLock.TryLock()
	if locked != false || err != nil {
		t.Fatalf("should have failed locking: locked: %t, err: %v", locked, err)
	}

	if newLock.fh != nil {
		t.Fatal("file handle should have been released and be nil")
	}
}
