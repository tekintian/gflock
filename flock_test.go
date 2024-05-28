// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package gflock_test

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/tekintian/gflock"

	. "gopkg.in/check.v1"
)

type TestSuite struct {
	path   string
	gflock *gflock.GFlock
}

var _ = Suite(&TestSuite{})

func Test(t *testing.T) { TestingT(t) }

func (t *TestSuite) SetUpTest(c *C) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "gflock-")
	c.Assert(err, IsNil)
	c.Assert(tmpFile, Not(IsNil))

	t.path = tmpFile.Name()

	defer os.Remove(t.path)
	tmpFile.Close()

	t.gflock = gflock.New(t.path)
}

func (t *TestSuite) TearDownTest(c *C) {
	t.gflock.Unlock()
	os.Remove(t.path)
}

func (t *TestSuite) TestNew(c *C) {
	f := gflock.New(t.path)
	c.Assert(f, Not(IsNil))
	c.Check(f.Path(), Equals, t.path)
	c.Check(f.Locked(), Equals, false)
	c.Check(f.RLocked(), Equals, false)
}

func (t *TestSuite) TestGFlockPath(c *C) {
	path := t.gflock.Path()
	c.Check(path, Equals, t.path)
}

func (t *TestSuite) TestGFlockLocked(c *C) {
	locked := t.gflock.Locked()
	c.Check(locked, Equals, false)
}

func (t *TestSuite) TestGFlockRLocked(c *C) {
	locked := t.gflock.RLocked()
	c.Check(locked, Equals, false)
}

func (t *TestSuite) TestGFlockString(c *C) {
	str := t.gflock.String()
	c.Assert(str, Equals, t.path)
}

func (t *TestSuite) TestGFlockTryLock(c *C) {
	c.Assert(t.gflock.Locked(), Equals, false)
	c.Assert(t.gflock.RLocked(), Equals, false)

	var locked bool
	var err error

	locked, err = t.gflock.TryLock()
	c.Assert(err, IsNil)
	c.Check(locked, Equals, true)
	c.Check(t.gflock.Locked(), Equals, true)
	c.Check(t.gflock.RLocked(), Equals, false)

	locked, err = t.gflock.TryLock()
	c.Assert(err, IsNil)
	c.Check(locked, Equals, true)

	// make sure we just return false with no error in cases
	// where we would have been blocked
	locked, err = gflock.New(t.path).TryLock()
	c.Assert(err, IsNil)
	c.Check(locked, Equals, false)
}

func (t *TestSuite) TestGFlockTryRLock(c *C) {
	c.Assert(t.gflock.Locked(), Equals, false)
	c.Assert(t.gflock.RLocked(), Equals, false)

	var locked bool
	var err error

	locked, err = t.gflock.TryRLock()
	c.Assert(err, IsNil)
	c.Check(locked, Equals, true)
	c.Check(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, true)

	locked, err = t.gflock.TryRLock()
	c.Assert(err, IsNil)
	c.Check(locked, Equals, true)

	// shared lock should not block.
	flock2 := gflock.New(t.path)
	locked, err = flock2.TryRLock()
	c.Assert(err, IsNil)
	if runtime.GOOS == "aix" {
		// When using POSIX locks, we can't safely read-lock the same
		// inode through two different descriptors at the same time:
		// when the first descriptor is closed, the second descriptor
		// would still be open but silently unlocked. So a second
		// TryRLock must return false.
		c.Check(locked, Equals, false)
	} else {
		c.Check(locked, Equals, true)
	}

	// make sure we just return false with no error in cases
	// where we would have been blocked
	t.gflock.Unlock()
	flock2.Unlock()
	t.gflock.Lock()
	locked, err = gflock.New(t.path).TryRLock()
	c.Assert(err, IsNil)
	c.Check(locked, Equals, false)
}

func (t *TestSuite) TestGFlockTryLockContext(c *C) {
	// happy path
	ctx, cancel := context.WithCancel(context.Background())
	locked, err := t.gflock.TryLockContext(ctx, time.Second)
	c.Assert(err, IsNil)
	c.Check(locked, Equals, true)

	// context already canceled
	cancel()
	locked, err = gflock.New(t.path).TryLockContext(ctx, time.Second)
	c.Assert(err, Equals, context.Canceled)
	c.Check(locked, Equals, false)

	// timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	locked, err = gflock.New(t.path).TryLockContext(ctx, time.Second)
	c.Assert(err, Equals, context.DeadlineExceeded)
	c.Check(locked, Equals, false)
}

func (t *TestSuite) TestGFlockTryRLockContext(c *C) {
	// happy path
	ctx, cancel := context.WithCancel(context.Background())
	locked, err := t.gflock.TryRLockContext(ctx, time.Second)
	c.Assert(err, IsNil)
	c.Check(locked, Equals, true)

	// context already canceled
	cancel()
	locked, err = gflock.New(t.path).TryRLockContext(ctx, time.Second)
	c.Assert(err, Equals, context.Canceled)
	c.Check(locked, Equals, false)

	// timeout
	t.gflock.Unlock()
	t.gflock.Lock()
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	locked, err = gflock.New(t.path).TryRLockContext(ctx, time.Second)
	c.Assert(err, Equals, context.DeadlineExceeded)
	c.Check(locked, Equals, false)
}

func (t *TestSuite) TestGFlockUnlock(c *C) {
	var err error

	err = t.gflock.Unlock()
	c.Assert(err, IsNil)

	// get a lock for us to unlock
	locked, err := t.gflock.TryLock()
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, true)
	c.Assert(t.gflock.Locked(), Equals, true)
	c.Check(t.gflock.RLocked(), Equals, false)

	_, err = os.Stat(t.path)
	c.Assert(os.IsNotExist(err), Equals, false)

	err = t.gflock.Unlock()
	c.Assert(err, IsNil)
	c.Check(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, false)
}

func (t *TestSuite) TestGFlockLock(c *C) {
	c.Assert(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, false)

	var err error

	err = t.gflock.Lock()
	c.Assert(err, IsNil)
	c.Check(t.gflock.Locked(), Equals, true)
	c.Check(t.gflock.RLocked(), Equals, false)

	// test that the short-circuit works
	err = t.gflock.Lock()
	c.Assert(err, IsNil)

	//
	// Test that Lock() is a blocking call
	//
	ch := make(chan error, 2)
	gf := gflock.New(t.path)
	defer gf.Unlock()

	go func(ch chan<- error) {
		ch <- nil
		ch <- gf.Lock()
		close(ch)
	}(ch)

	errCh, ok := <-ch
	c.Assert(ok, Equals, true)
	c.Assert(errCh, IsNil)

	err = t.gflock.Unlock()
	c.Assert(err, IsNil)

	errCh, ok = <-ch
	c.Assert(ok, Equals, true)
	c.Assert(errCh, IsNil)
	c.Check(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, false)
	c.Check(gf.Locked(), Equals, true)
	c.Check(gf.RLocked(), Equals, false)
}

func (t *TestSuite) TestGFlockRLock(c *C) {
	c.Assert(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, false)

	var err error

	err = t.gflock.RLock()
	c.Assert(err, IsNil)
	c.Check(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, true)

	// test that the short-circuit works
	err = t.gflock.RLock()
	c.Assert(err, IsNil)

	//
	// Test that RLock() is a blocking call
	//
	ch := make(chan error, 2)
	gf := gflock.New(t.path)
	defer gf.Unlock()

	go func(ch chan<- error) {
		ch <- nil
		ch <- gf.RLock()
		close(ch)
	}(ch)

	errCh, ok := <-ch
	c.Assert(ok, Equals, true)
	c.Assert(errCh, IsNil)

	err = t.gflock.Unlock()
	c.Assert(err, IsNil)

	errCh, ok = <-ch
	c.Assert(ok, Equals, true)
	c.Assert(errCh, IsNil)
	c.Check(t.gflock.Locked(), Equals, false)
	c.Check(t.gflock.RLocked(), Equals, false)
	c.Check(gf.Locked(), Equals, false)
	c.Check(gf.RLocked(), Equals, true)
}
