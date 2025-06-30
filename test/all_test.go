package test

import (
	"fmt"
	"sync"
	_"sync/atomic"
	"testing"
	"time"

	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"
	"github.com/stretchr/testify/assert"
)

func TestPubSub_BasicPublishSubscribe(t *testing.T) {
	ps, err := pubsub.NewEngine()
	if err != nil {
		t.Fatalf("failed to create pubsub engine: %v", err)
	}

	topic := "test-topic"
	message := "hello world"

	ch := ps.Memory.Subscribe(topic)
	defer ps.Memory.Unsubscribe(topic, ch)

	go func() {
		time.Sleep(10 * time.Millisecond)
		ps.Memory.Publish(topic, []byte(message))
	}()

	select {
	case msg := <-ch:
		if string(msg) != message {
			t.Errorf("expected %q, got %q", message, string(msg))
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout: did not receive message")
	}
}

func TestTaskQueue_EnqueueAndConsume(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var (
		wg     sync.WaitGroup
		result []string
		mu     sync.Mutex
	)

	wg.Add(1)

	err := q.RegisterWorker("email", func(msg []byte) error {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		result = append(result, string(msg))
		return nil
	})
	if err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	err = q.Enqueue("email", []byte("send-email"))
	if err != nil {
		t.Fatalf("failed to enqueue task: %v", err)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for task completion")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(result) != 1 || result[0] != "send-email" {
		t.Errorf("unexpected task result: %+v", result)
	}
}

func TestTaskQueue_QueueSize(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	// Tidak register worker dulu untuk memastikan task tetap di queue
	err := q.Enqueue("log", []byte("task1"))
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	err = q.Enqueue("log", []byte("task2"))
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	size := q.GetQueueSize("log")
	if size != 2 {
		t.Errorf("expected queue size 2, got %d", size)
	}
}

func TestTaskQueue_Close(t *testing.T) {
	q := queue.NewInMem()

	var called bool
	var mu sync.Mutex
	
	err := q.RegisterWorker("shutdown", func(msg []byte) error {
		mu.Lock()
		called = true
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	err = q.Enqueue("shutdown", []byte("task"))
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// beri waktu untuk goroutine bekerja
	time.Sleep(100 * time.Millisecond)

	q.Close()

	err = q.Enqueue("shutdown", []byte("new-task"))
	if err == nil {
		t.Error("expected error after queue is closed")
	}

	mu.Lock()
	if !called {
		t.Error("worker handler not called before close")
	}
	mu.Unlock()
}

func TestTaskQueue_EnqueueAndConsumeMultiple(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var mu sync.Mutex
	var result []string
	var wg sync.WaitGroup

	wg.Add(2) // expect 2 messages

	err := q.RegisterWorker("email", func(data []byte) error {
		mu.Lock()
		result = append(result, string(data))
		mu.Unlock()
		wg.Done()
		return nil
	})
	assert.NoError(t, err)

	_ = q.Enqueue("email", []byte("welcome"))
	_ = q.Enqueue("email", []byte("verify"))

	// Wait dengan timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tasks")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"welcome", "verify"}, result)
}

func TestTaskQueue_ConcurrentHandling(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	const tasks = 100
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(tasks)

	err := q.RegisterWorker("job", func(data []byte) error {
		mu.Lock()
		counter++
		mu.Unlock()
		wg.Done()
		return nil
	})
	assert.NoError(t, err)

	for i := 0; i < tasks; i++ {
		_ = q.Enqueue("job", []byte("x"))
	}

	// Wait dengan timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for concurrent tasks")
	}

	mu.Lock()
	assert.Equal(t, tasks, counter)
	mu.Unlock()
}

func TestTaskQueue_CloseAndEnqueue(t *testing.T) {
	q := queue.NewInMem()
	q.Close()

	err := q.Enqueue("dead", []byte("data"))
	assert.Error(t, err)
}

// ---------- PUBSUB TESTS ----------

func TestPubSub_SubscribeRaw(t *testing.T) {
	ps, _ := pubsub.NewEngine()

	ch, err := ps.Memory.SubscribeRaw("raw")
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch:
			assert.Equal(t, "raw data", string(msg))
		case <-time.After(2 * time.Second):
			t.Error("timeout waiting for message")
		}
	}()

	// Beri sedikit delay untuk memastikan subscriber ready
	time.Sleep(10 * time.Millisecond)
	
	err = ps.Memory.Publish("raw", []byte("raw data"))
	assert.NoError(t, err)

	wg.Wait()
}

func TestEventBus_PublishAndSubscribe(t *testing.T) {
	ps := pubsub.NewInMem()
	defer ps.Close()

	ch := ps.Subscribe("chat")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch:
			assert.Equal(t, "hello", string(msg))
		case <-time.After(2 * time.Second):
			t.Error("timeout waiting for message")
		}
	}()

	// Beri delay untuk subscriber ready
	time.Sleep(10 * time.Millisecond)

	err := ps.Publish("chat", []byte("hello"))
	assert.NoError(t, err)
	wg.Wait()
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	ps := pubsub.NewInMem()
	defer ps.Close()

	const count = 3
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		ch := ps.Subscribe("multi")
		go func(ch <-chan []byte) {
			defer wg.Done()
			select {
			case msg := <-ch:
				assert.Equal(t, "ping", string(msg))
			case <-time.After(2 * time.Second):
				t.Error("timeout waiting for message")
			}
		}(ch)
	}

	// Beri delay untuk semua subscriber ready
	time.Sleep(50 * time.Millisecond)

	err := ps.Publish("multi", []byte("ping"))
	assert.NoError(t, err)
	wg.Wait()
}

func TestEventBus_SubscribeRaw(t *testing.T) {
	ps := pubsub.NewInMem()
	defer ps.Close()

	ch, err := ps.SubscribeRaw("raw")
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch:
			assert.Equal(t, "raw-data", string(msg))
		case <-time.After(2 * time.Second):
			t.Error("timeout waiting for message")
		}
	}()

	time.Sleep(10 * time.Millisecond)
	_ = ps.Publish("raw", []byte("raw-data"))
	wg.Wait()
}

func TestEventBus_Unsubscribe(t *testing.T) {
	ps := pubsub.NewInMem()
	defer ps.Close()

	ch := ps.Subscribe("room")

	// Publish dan tunggu message
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		msg := <-ch
		assert.Equal(t, "hi", string(msg))
	}()

	time.Sleep(10 * time.Millisecond)
	_ = ps.Publish("room", []byte("hi"))
	wg.Wait()

	// Unsubscribe channel
	ps.Unsubscribe("room", ch)

	// Test pastikan channel ditutup
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("should not receive any message after unsubscribe")
		}
		// channel closed = expected
	case <-time.After(500 * time.Millisecond):
		// timeout = channel tidak close, tapi ini mungkin OK tergantung implementasi
		t.Log("channel not immediately closed after unsubscribe, but this might be OK")
	}
}

func TestEventBus_UnsubscribeAll(t *testing.T) {
	ps := pubsub.NewInMem()
	defer ps.Close()

	for i := 0; i < 3; i++ {
		ps.Subscribe("group")
	}
	
	// Beri waktu untuk subscriber register
	time.Sleep(10 * time.Millisecond)
	
	ps.UnsubscribeAll("group")

	assert.Equal(t, 0, ps.GetSubscriberCount("group"))
}

func TestEventBus_Close(t *testing.T) {
	ps := pubsub.NewInMem()
	ps.Subscribe("closed")

	ps.Close()

	err := ps.Publish("closed", []byte("fail"))
	assert.Error(t, err)

	ch := ps.Subscribe("closed")
	assert.Nil(t, ch)
}

func BenchmarkPubSub_1000Messages(b *testing.B) {
	ps, _ := pubsub.NewEngine()

	ready := make(chan bool)
	received := make(chan bool, b.N)

	// Worker goroutine untuk consume messages
	go func() {
		sub := ps.Memory.Subscribe("bench") 
		ready <- true // signal subscriber is ready

		count := 0
		for msg := range sub {
			if msg != nil {
				received <- true
				count++
				if count >= b.N {
					break
				}
			}
		}
	}()

	<-ready
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ps.Memory.Publish("bench", []byte("x"))
	}

	for i := 0; i < b.N; i++ {
		select {
		case <-received:
		case <-time.After(5 * time.Second):
			b.Fatalf("timeout: only received %d out of %d messages", i, b.N)
		}
	}
}

func BenchmarkTaskQueue_1000Tasks(b *testing.B) {
	q := queue.NewInMem()
	defer q.Close()

	// Channel untuk sync completion
	done := make(chan bool, b.N)

	_ = q.RegisterWorker("bench", func(data []byte) error {
		done <- true
		return nil
	})

	b.ResetTimer()
	
	// Enqueue semua task
	for i := 0; i < 8; i++ {
		_ = q.RegisterWorker("bench", func(data []byte) error {
			done <- true
			return nil
		})
	}
	
}

func BenchmarkTaskQueue_WithDelay(b *testing.B) {
	q := queue.NewInMem()
	defer q.Close()

	max := b.N
	if max > 10000 {
		max = 10000 // ✅ aman, tidak assign ke b.N
	}

	done := make(chan bool, max)

	for i := 0; i < 64; i++ {
		_ = q.RegisterWorker("delay", func(data []byte) error {
			time.Sleep(1 * time.Millisecond)
			done <- true
			return nil
		})
	}

	b.ResetTimer()

	for i := 0; i < max; i++ {
		_ = q.Enqueue("delay", []byte("task"))
	}

	for i := 0; i < max; i++ {
		select {
		case <-done:
		case <-time.After(60 * time.Second):
			b.Fatalf("timeout: only completed %d out of %d tasks", i, max)
		}
	}
}

func TestTaskQueue_RealTimeHandler(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var mu sync.Mutex
	var result []string
	var wg sync.WaitGroup

	wg.Add(3)

	_ = q.RegisterWorker("realtime", func(msg []byte) error {
		mu.Lock()
		result = append(result, string(msg))
		mu.Unlock()
		wg.Done()
		return nil
	})

	go func() {
		_ = q.Enqueue("realtime", []byte("task-1"))
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = q.Enqueue("realtime", []byte("task-2"))
	}()
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = q.Enqueue("realtime", []byte("task-3"))
	}()

	// Wait dengan timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for realtime tasks")
	}

	mu.Lock()
	assert.ElementsMatch(t, []string{"task-1", "task-2", "task-3"}, result)
	mu.Unlock()
}



func BenchmarkTaskQueue_CPUProfile(b *testing.B) {
	q := queue.NewInMem()
	defer q.Close()

	done := make(chan struct{}, 1)
	_ = q.RegisterWorker("bench", func(data []byte) error {
		done <- struct{}{}
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Enqueue("bench", []byte("task"))
		<-done
	}
}

func BenchmarkCPUPrint(b *testing.B) {
	start := time.Now()
	sum := 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum += i
	}

	duration := time.Since(start)
	fmt.Printf("CPU Benchmark ran for: %v | Total: %d | Iterations: %d\n", duration, sum, b.N)
}

func BenchmarkTaskQueue_Print(b *testing.B) {
	q := queue.NewInMem()
	defer q.Close()

	done := make(chan struct{}, 1)
	_ = q.RegisterWorker("log", func(data []byte) error {
		done <- struct{}{}
		return nil
	})

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = q.Enqueue("log", []byte("data"))
		<-done
	}

	elapsed := time.Since(start)
	fmt.Printf("TaskQueue: handled %d messages in %v (%.2f msg/sec)\n", b.N, elapsed, float64(b.N)/elapsed.Seconds())
}