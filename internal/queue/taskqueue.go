package queue

import (
	"context"
	"errors"
	"sync"
)

type TaskQueue struct {
	mu      sync.RWMutex
	queues  map[string]chan []byte
	workers map[string][]context.CancelFunc
	closed  bool
}

func NewInMem() *TaskQueue {
	return &TaskQueue{
		queues:  make(map[string]chan []byte),
		workers: make(map[string][]context.CancelFunc),
		closed:  false,
	}
}

func (q *TaskQueue) Enqueue(queue string, task []byte) error {
	q.mu.RLock()
	ch, ok := q.queues[queue]
	closed := q.closed
	q.mu.RUnlock()

	if closed {
		return errors.New("task queue is closed")
	}

	if !ok {
		q.mu.Lock()
		ch, ok = q.queues[queue]
		if !ok {
			ch = make(chan []byte, 10000) // ✅ buffer besar agar tidak deadlock
			q.queues[queue] = ch
		}
		q.mu.Unlock()
	}

	select {
	case ch <- task:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func (q *TaskQueue) RegisterWorker(queue string, handler func([]byte) error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return errors.New("task queue is closed")
	}
	ch, ok := q.queues[queue]
	if !ok {
		ch = make(chan []byte, 10000) // ✅ buffer besar agar tidak deadlock saat enqueue masif
		q.queues[queue] = ch
	}

	// Daftarkan context cancel, agar bisa graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	q.workers[queue] = append(q.workers[queue], cancel)

	// Jalankan worker dalam goroutine
	go func() {
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return // channel ditutup
				}
				_ = handler(msg) // abaikan error untuk sekarang
			case <-ctx.Done():
				return // shutdown
			}
		}
	}()

	return nil
}


func (q *TaskQueue) GetQueueSize(queue string) int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	if ch, ok := q.queues[queue]; ok {
		return len(ch)
	}
	return 0
}

func (q *TaskQueue) GetQueues() []string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	queues := make([]string, 0, len(q.queues))
	for name := range q.queues {
		queues = append(queues, name)
	}
	return queues
}

func (q *TaskQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.closed = true
	
	for queue, cancels := range q.workers {
		for _, cancel := range cancels {
			cancel()
		}
		delete(q.workers, queue)
	}
	
	for queue, ch := range q.queues {
		close(ch)
		delete(q.queues, queue)
	}
}
