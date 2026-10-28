package filelock

import (
	"errors"
	"testing"
	"time"
)

func TestLockExcludesAndLetsGo(t *testing.T) {
	dir := t.TempDir()
	unlock, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(dir, 20*time.Millisecond); !errors.Is(err, ErrTimeout) {
		t.Fatalf("a second Lock while held: err = %v, want ErrTimeout", err)
	}
	unlock()
	again, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatalf("Lock after unlock: %v", err)
	}
	again()
}

func TestLockWaitsForTheHolder(t *testing.T) {
	dir := t.TempDir()
	unlock, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		unlock()
	}()
	second, err := Lock(dir, 5*time.Second)
	if err != nil {
		t.Fatalf("the waiter did not get the lock: %v", err)
	}
	second()
}
