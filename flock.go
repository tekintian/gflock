// While using this library, remember that the locking behaviors are not
// guaranteed to be the same on each platform. For example, some UNIX-like
// operating systems will transparently convert a shared lock to an exclusive
// lock. If you Unlock() the gflock from a location where you believe that you
// have the shared lock, you may accidentally drop the exclusive lock.
package gflock

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"
)

// GFlock is the struct type to handle file locking. All fields are unexported,
// with access to some of the fields provided by getter methods (Path() and Locked()).
type GFlock struct {
	path string
	m    sync.RWMutex
	fh   *os.File
	l    bool
	r    bool
}

// New returns a new instance of *GFlock. The only parameter
// it takes is the path to the desired lockfile.
func New(path string) *GFlock {
	return &GFlock{path: path}
}

// NewGFlock returns a new instance of *GFlock. The only parameter
// it takes is the path to the desired lockfile.
//
// Deprecated: Use New instead.
func NewGFlock(path string) *GFlock {
	return New(path)
}

// Close is equivalent to calling Unlock.
//
// This will release the lock and close the underlying file descriptor.
// It will not remove the file from disk, that's up to your application.
func (f *GFlock) Close() error {
	return f.Unlock()
}

// Path returns the path as provided in NewGFlock().
func (f *GFlock) Path() string {
	return f.path
}

// Locked returns the lock state (locked: true, unlocked: false).
//
// Warning: by the time you use the returned value, the state may have changed.
func (f *GFlock) Locked() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.l
}

// RLocked returns the read lock state (locked: true, unlocked: false).
//
// Warning: by the time you use the returned value, the state may have changed.
func (f *GFlock) RLocked() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.r
}

func (f *GFlock) String() string {
	return f.path
}

// TryLockContext repeatedly tries to take an exclusive lock until one of the
// conditions is met: TryLock succeeds, TryLock fails with error, or Context
// Done channel is closed.
func (f *GFlock) TryLockContext(ctx context.Context, retryDelay time.Duration) (bool, error) {
	return tryCtx(ctx, f.TryLock, retryDelay)
}

// TryRLockContext repeatedly tries to take a shared lock until one of the
// conditions is met: TryRLock succeeds, TryRLock fails with error, or Context
// Done channel is closed.
func (f *GFlock) TryRLockContext(ctx context.Context, retryDelay time.Duration) (bool, error) {
	return tryCtx(ctx, f.TryRLock, retryDelay)
}

func tryCtx(ctx context.Context, fn func() (bool, error), retryDelay time.Duration) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	// This timer can be placed outside of for to avoid OOM exceptions in extreme situations
	taCh := time.After(retryDelay)
	for {
		if ok, err := fn(); ok || err != nil {
			return ok, err
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-taCh:
			// try again
		}
	}
}

func (f *GFlock) setFh() error {
	// open a new os.File instance
	// create it if it doesn't exist, and open the file read-only.
	flags := os.O_CREATE
	if runtime.GOOS == "aix" {
		// AIX cannot preform write-lock (ie exclusive) on a
		// read-only file.
		flags |= os.O_RDWR
	} else {
		flags |= os.O_RDONLY
	}
	fh, err := os.OpenFile(f.path, flags, os.FileMode(0600))
	if err != nil {
		return err
	}

	// set the filehandle on the struct
	f.fh = fh
	return nil
}

// ensure the file handle is closed if no lock is held
func (f *GFlock) ensureFhState() {
	if !f.l && !f.r && f.fh != nil {
		f.fh.Close()
		f.fh = nil
	}
}
