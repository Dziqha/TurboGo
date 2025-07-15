package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
)

type TaskQueue struct {
	mu                   sync.RWMutex
	queues               map[string]chan []byte
	workers              map[string][]context.CancelFunc
	closed               bool
	AllowMultipleWorkers bool // ✅ optional: true = banyak worker per queue
}

func NewInMem() *TaskQueue {
	return &TaskQueue{
		queues:  make(map[string]chan []byte),
		workers: make(map[string][]context.CancelFunc),
	}
}

func (q *TaskQueue) Enqueue(queue string, task []byte) error {
	if strings.TrimSpace(queue) == "" {
		return errors.New("queue name is required")
	}

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
	if strings.TrimSpace(queue) == "" {
		return errors.New("queue name is required")
	}
	if !q.AllowMultipleWorkers {
		if _, exists := q.workers[queue]; exists {
			return fmt.Errorf("worker for queue %q already exists", queue)
		}
	}

	ch, ok := q.queues[queue]
	if !ok {
		ch = make(chan []byte, 10000)
		q.queues[queue] = ch
	}

	ctx, cancel := context.WithCancel(context.Background())
	q.workers[queue] = append(q.workers[queue], cancel) // ✅ now supports multiple workers if allowed

	go func() {
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				_ = safeHandler(handler)(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func safeHandler(handler func([]byte) error) func([]byte) error {
	return func(msg []byte) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Worker Panic] recovered: %v", r)
			}
		}()
		return handler(msg)
	}
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
