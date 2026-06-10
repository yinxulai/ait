package queue

import (
	"errors"
	"testing"
)

func TestQueueFIFO(t *testing.T) {
	q := New[int](2)
	if err := q.Enqueue(1); err != nil {
		t.Fatalf("enqueue 1: %v", err)
	}
	if err := q.Enqueue(2); err != nil {
		t.Fatalf("enqueue 2: %v", err)
	}
	q.Close()

	var got []int
	for item := range q.Items() {
		got = append(got, item)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("got %v, want [1 2]", got)
	}
}

func TestQueueFull(t *testing.T) {
	q := New[int](1)
	if err := q.Enqueue(1); err != nil {
		t.Fatalf("enqueue 1: %v", err)
	}
	if err := q.Enqueue(2); !errors.Is(err, ErrFull) {
		t.Fatalf("enqueue full: got %v, want %v", err, ErrFull)
	}
}

func TestQueueEnqueueUntilDone(t *testing.T) {
	q := New[int](0)
	done := make(chan struct{})
	close(done)
	if err := q.EnqueueUntil(done, 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("enqueue until done: got %v, want %v", err, ErrClosed)
	}
}

func TestQueueClosed(t *testing.T) {
	q := New[int](1)
	q.Close()
	if err := q.Enqueue(1); !errors.Is(err, ErrClosed) {
		t.Fatalf("enqueue closed: got %v, want %v", err, ErrClosed)
	}
}
