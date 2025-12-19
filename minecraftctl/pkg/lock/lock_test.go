package lock

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewFileLock(t *testing.T) {
	fl := NewFileLock("/tmp/test.lock")
	if fl == nil {
		t.Fatal("NewFileLock returned nil")
	}
	if fl.path != "/tmp/test.lock" {
		t.Errorf("path = %q, want /tmp/test.lock", fl.path)
	}
	if fl.file != nil {
		t.Error("file should be nil before locking")
	}
}

func TestLockAndUnlock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	fl := NewFileLock(lockPath)

	// Acquire lock
	err := fl.Lock()
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	// Verify lock file was created
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("Lock file not created: %v", err)
	}

	// Verify file handle is open
	if fl.file == nil {
		t.Error("file handle should not be nil after locking")
	}

	// Release lock
	err = fl.Unlock()
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Verify file handle is closed
	if fl.file != nil {
		t.Error("file handle should be nil after unlocking")
	}
}

func TestUnlockWithoutLock(t *testing.T) {
	fl := NewFileLock("/tmp/nonexistent.lock")

	// Unlock without lock should not error
	err := fl.Unlock()
	if err != nil {
		t.Errorf("Unlock without lock should not error: %v", err)
	}
}

func TestDoubleUnlock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	fl := NewFileLock(lockPath)
	fl.Lock()
	fl.Unlock()

	// Second unlock should not error
	err := fl.Unlock()
	if err != nil {
		t.Errorf("Double unlock should not error: %v", err)
	}
}

func TestNonBlockingLock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// First lock
	fl1 := NewFileLock(lockPath)
	err := fl1.Lock()
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	defer fl1.Unlock()

	// Second lock should fail immediately with non-blocking
	fl2 := NewFileLock(lockPath)
	err = fl2.LockWithOptions(LockOptions{NonBlocking: true})
	if err == nil {
		fl2.Unlock()
		t.Error("Non-blocking lock should fail when lock is held")
	}
}

func TestLockTimeout(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// First lock
	fl1 := NewFileLock(lockPath)
	err := fl1.Lock()
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	defer fl1.Unlock()

	// Second lock should timeout
	fl2 := NewFileLock(lockPath)
	start := time.Now()
	err = fl2.TryLock(200 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		fl2.Unlock()
		t.Error("Lock should have timed out")
	}

	// Should have waited approximately the timeout duration
	if elapsed < 150*time.Millisecond {
		t.Errorf("Timeout was too short: %v", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("Timeout was too long: %v", elapsed)
	}
}

func TestLockReleasedBeforeTimeout(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// First lock
	fl1 := NewFileLock(lockPath)
	err := fl1.Lock()
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}

	// Release lock after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		fl1.Unlock()
	}()

	// Second lock should succeed before timeout
	fl2 := NewFileLock(lockPath)
	start := time.Now()
	err = fl2.TryLock(1 * time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Lock should have succeeded: %v", err)
	} else {
		fl2.Unlock()
	}

	// Should have acquired lock relatively quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("Lock took too long to acquire: %v", elapsed)
	}
}

func TestConcurrentLocking(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	var counter int64
	var wg sync.WaitGroup
	const numGoroutines = 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			fl := NewFileLock(lockPath)
			if err := fl.Lock(); err != nil {
				t.Errorf("Lock failed: %v", err)
				return
			}

			// Critical section - increment counter
			// Use atomic operations to satisfy the race detector while still
			// testing that file locks serialize access (if the lock fails,
			// goroutines would interleave read-sleep-write and counter would be wrong)
			current := atomic.LoadInt64(&counter)
			time.Sleep(10 * time.Millisecond) // Simulate work
			atomic.StoreInt64(&counter, current+1)

			fl.Unlock()
		}()
	}

	wg.Wait()

	// If locking worked correctly, counter should be exactly numGoroutines
	finalCount := atomic.LoadInt64(&counter)
	if finalCount != numGoroutines {
		t.Errorf("Counter = %d, want %d (race condition detected)", finalCount, numGoroutines)
	}
}

func TestLockInvalidPath(t *testing.T) {
	// Try to lock a path in a non-existent directory
	fl := NewFileLock("/nonexistent/directory/test.lock")
	err := fl.Lock()
	if err == nil {
		fl.Unlock()
		t.Error("Lock should fail for invalid path")
	}
}

func TestLockOptionsStruct(t *testing.T) {
	opts := LockOptions{
		Timeout:     5 * time.Second,
		NonBlocking: true,
	}

	if opts.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", opts.Timeout)
	}
	if !opts.NonBlocking {
		t.Error("NonBlocking should be true")
	}
}

func TestFileLockStruct(t *testing.T) {
	fl := &FileLock{
		path: "/test/path",
		file: nil,
	}

	if fl.path != "/test/path" {
		t.Errorf("path = %q, want /test/path", fl.path)
	}
}

func TestReacquireLockAfterUnlock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	fl := NewFileLock(lockPath)

	// First lock/unlock cycle
	if err := fl.Lock(); err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	if err := fl.Unlock(); err != nil {
		t.Fatalf("First unlock failed: %v", err)
	}

	// Second lock/unlock cycle with same FileLock
	if err := fl.Lock(); err != nil {
		t.Fatalf("Second lock failed: %v", err)
	}
	if err := fl.Unlock(); err != nil {
		t.Fatalf("Second unlock failed: %v", err)
	}
}

func TestMultipleLockInstances(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// Create multiple FileLock instances for the same path
	fl1 := NewFileLock(lockPath)
	fl2 := NewFileLock(lockPath)

	// Lock with first instance
	if err := fl1.Lock(); err != nil {
		t.Fatalf("First instance lock failed: %v", err)
	}

	// Second instance should fail with non-blocking
	err := fl2.LockWithOptions(LockOptions{NonBlocking: true})
	if err == nil {
		fl2.Unlock()
		t.Error("Second instance should fail to lock")
	}

	// Release first lock
	fl1.Unlock()

	// Now second instance should succeed
	if err := fl2.Lock(); err != nil {
		t.Fatalf("Second instance should succeed after first unlocks: %v", err)
	}
	fl2.Unlock()
}

func TestZeroTimeout(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	fl := NewFileLock(lockPath)

	// Zero timeout should work (means block forever, but we can test uncontested lock)
	err := fl.TryLock(0)
	if err != nil {
		t.Fatalf("TryLock(0) failed: %v", err)
	}
	fl.Unlock()
}
