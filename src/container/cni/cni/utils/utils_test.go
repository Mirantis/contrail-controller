package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestDoWithRetriesError(t *testing.T) {
	var executed int
	err := DoWithRetries(5, 0*time.Millisecond, func() error {
		executed++
		return fmt.Errorf("executed %d", executed)
	})
	if err == nil {
		t.Fatal("expected error != nil")
	}
	if executed != 5 {
		t.Fatal("function supposed to be executed maximum amount of times")
	}
}

func TestDoWithRetriesSuccess(t *testing.T) {
	var executed int
	err := DoWithRetries(5, 0*time.Millisecond, func() error {
		executed++
		return nil
	})
	if err != nil {
		t.Fatalf("error expected to be nil: %v", err)
	}
	if executed != 1 {
		t.Fatal("function supposed to be executed only once")
	}
}
