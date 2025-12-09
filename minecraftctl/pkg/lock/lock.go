package lock

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// FileLock manages file-based locking using flock
type FileLock struct {
	path string
	file *os.File
}

// LockOptions controls lock acquisition behavior
type LockOptions struct {
	Timeout     time.Duration // 0 = block forever
	NonBlocking bool          // Try once and fail immediately
}

// NewFileLock creates a new file lock at the specified path
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// Lock acquires an exclusive lock, blocking until available
func (fl *FileLock) Lock() error {
	return fl.TryLock(0)
}

// TryLock attempts to acquire an exclusive lock with optional timeout
func (fl *FileLock) TryLock(timeout time.Duration) error {
	return fl.LockWithOptions(LockOptions{
		Timeout:     timeout,
		NonBlocking: false,
	})
}

// LockWithOptions acquires a lock with the specified options
func (fl *FileLock) LockWithOptions(opts LockOptions) error {
	// Open/create lock file
	file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	fl.file = file

	// Set up signal handlers to release lock on termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fl.Unlock()
		os.Exit(1)
	}()

	if opts.NonBlocking {
		// Try once and return immediately
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			file.Close()
			return fmt.Errorf("lock is held by another process")
		}
		return nil
	}

	// Attempt to acquire exclusive lock with optional timeout
	start := time.Now()
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Lock acquired
			return nil
		}

		if opts.Timeout > 0 && time.Since(start) >= opts.Timeout {
			file.Close()
			fl.file = nil
			return fmt.Errorf("timeout waiting for lock after %v", opts.Timeout)
		}

		// Retry after a short delay
		time.Sleep(100 * time.Millisecond)
	}
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
	closeErr := fl.file.Close()
	fl.file = nil

	if err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}
	return closeErr
}
