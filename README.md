# go file lock

Go 语言里面用的文件锁 gflock, 来源于 gofrs/flock,做了一系列升级改造和 Bug 修复!

`gflock` implements a thread-safe sync.Locker interface for file locking. It also
includes a non-blocking TryLock() function to allow locking without blocking execution.

## Installation

```
go get -u github.com/tekitian/gflock
```

## Usage

```Go
import "github.com/tekitian/gflock"

fileLock := gflock.New("/var/lock/go-lock.lock")

locked, err := fileLock.TryLock()

if err != nil {
	// handle locking error
}

if locked {
	// do work
	fileLock.Unlock()
}
```

## thanks for

https://github.com/gofrs/flock
