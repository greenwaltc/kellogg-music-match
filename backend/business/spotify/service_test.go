package spotify

import (
	"context"
	"testing"
	"time"
)

func TestServiceProgression(t *testing.T) {
	s := NewService(nil, "") // no persistence in unit test
	job := s.StartSync(context.Background(), "user1", "code123", "stateABC")
	if job.Status != StatusPending {
		t.Fatalf("expected pending, got %s", job.Status)
	}
	// Wait for completion (max ~5s)
	deadline := time.Now().Add(6 * time.Second)
	for {
		st := s.GetStatus("user1")
		if st.Status == StatusComplete && st.Progress == 100 {
			if st.FinishedAt == nil {
				t.Fatalf("finishedAt should be set")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for completion: %+v", st)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func TestRetrySyncBehavior(t *testing.T) {
	s := NewService(nil, "")
	// Start initial sync
	job := s.StartSync(context.Background(), "retryUser", "code1", "state1")
	if job.Status != StatusPending {
		t.Fatalf("expected pending, got %s", job.Status)
	}
	// Attempt retry while in progress should error
	// Wait briefly for job to transition to in_progress
	deadline := time.Now().Add(3 * time.Second)
	for {
		st := s.GetStatus("retryUser")
		if st.Status == StatusInProgress {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for in_progress, got %s", st.Status)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := s.RetrySync(context.Background(), "retryUser", "code2", "state2"); err == nil {
		t.Fatalf("expected error retrying while in progress")
	}
	// Wait for completion
	deadline = time.Now().Add(6 * time.Second)
	for {
		st := s.GetStatus("retryUser")
		if st.Status == StatusComplete {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for complete, got %s", st.Status)
		}
		time.Sleep(300 * time.Millisecond)
	}
	// Retry after completion should succeed
	job2, err := s.RetrySync(context.Background(), "retryUser", "code3", "state3")
	if err != nil {
		t.Fatalf("unexpected error on retry after completion: %v", err)
	}
	if job2.Status != StatusPending {
		t.Fatalf("expected pending on retry, got %s", job2.Status)
	}
}
