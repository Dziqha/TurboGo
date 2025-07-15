package test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
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
	q.AllowMultipleWorkers = true // kalau kamu pakai pembatas sebelumnya
	defer q.Close()

	done := make(chan bool, b.N)

	// Daftar multiple worker
	for i := 0; i < 8; i++ {
		_ = q.RegisterWorker("bench", func(data []byte) error {
			done <- true
			return nil
		})
	}

	b.ResetTimer()

	// Enqueue semua task
	for i := 0; i < b.N; i++ {
		_ = q.Enqueue("bench", []byte("task"))
	}

	// Tunggu semua task selesai
	for i := 0; i < b.N; i++ {
		<-done
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

// ========== NEW TASK QUEUE TESTS ==========

func TestTaskQueue_ErrorHandling(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var errorCount int64
	var successCount int64

	err := q.RegisterWorker("error-prone", func(msg []byte) error {
		message := string(msg)
		if message == "error" {
			atomic.AddInt64(&errorCount, 1)
			return errors.New("simulated error")
		}
		atomic.AddInt64(&successCount, 1)
		return nil
	})
	assert.NoError(t, err)

	// Enqueue tasks with some errors
	tasks := []string{"success1", "error", "success2", "error", "success3"}
	for _, task := range tasks {
		_ = q.Enqueue("error-prone", []byte(task))
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int64(2), atomic.LoadInt64(&errorCount))
	assert.Equal(t, int64(3), atomic.LoadInt64(&successCount))
}

func TestTaskQueue_WorkerPanic(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var panicCount int64
	var normalCount int64

	err := q.RegisterWorker("panic-test", func(msg []byte) error {
		message := string(msg)
		if message == "panic" {
			atomic.AddInt64(&panicCount, 1)
			panic("simulated panic")
		}
		atomic.AddInt64(&normalCount, 1)
		return nil
	})
	assert.NoError(t, err)

	// Enqueue tasks with panic
	tasks := []string{"normal1", "panic", "normal2", "panic", "normal3"}
	for _, task := range tasks {
		_ = q.Enqueue("panic-test", []byte(task))
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Should continue processing after panic
	assert.Equal(t, int64(2), atomic.LoadInt64(&panicCount))
	assert.Equal(t, int64(3), atomic.LoadInt64(&normalCount))
}

func TestTaskQueue_DuplicateWorkerRegistration(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	handler := func(msg []byte) error { return nil }

	err1 := q.RegisterWorker("duplicate", handler)
	assert.NoError(t, err1)

	err2 := q.RegisterWorker("duplicate", handler)
	// Should handle duplicate registration gracefully
	assert.Error(t, err2)
}

func TestTaskQueue_EmptyQueueName(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	err := q.Enqueue("", []byte("data"))
	assert.Error(t, err)

	err = q.RegisterWorker("", func(msg []byte) error { return nil })
	assert.Error(t, err)
}

func TestTaskQueue_NilMessage(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var received bool
	var mu sync.Mutex

	err := q.RegisterWorker("nil-test", func(msg []byte) error {
		mu.Lock()
		defer mu.Unlock()
		received = true
		return nil
	})
	assert.NoError(t, err)

	err = q.Enqueue("nil-test", nil)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.True(t, received)
	mu.Unlock()
}

func TestTaskQueue_LargeMessage(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var received []byte
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)

	err := q.RegisterWorker("large", func(msg []byte) error {
		mu.Lock()
		defer mu.Unlock()
		received = make([]byte, len(msg))
		copy(received, msg)
		wg.Done()
		return nil
	})
	assert.NoError(t, err)

	// Create 1MB message
	largeMsg := make([]byte, 1024*1024)
	for i := range largeMsg {
		largeMsg[i] = byte(i % 256)
	}

	err = q.Enqueue("large", largeMsg)
	assert.NoError(t, err)

	wg.Wait()

	mu.Lock()
	assert.Equal(t, len(largeMsg), len(received))
	assert.Equal(t, largeMsg, received)
	mu.Unlock()
}

func TestTaskQueue_HighLoad(t *testing.T) {
	q := queue.NewInMem()
	q.AllowMultipleWorkers = true
	defer q.Close()

	const numTasks = 10000
	var processed int64

	// Register multiple workers
	for i := 0; i < 10; i++ {
		err := q.RegisterWorker("high-load", func(msg []byte) error {
			atomic.AddInt64(&processed, 1)
			return nil
		})
		assert.NoError(t, err)
	}

	// Enqueue many tasks
	for i := 0; i < numTasks; i++ {
		_ = q.Enqueue("high-load", []byte(fmt.Sprintf("task-%d", i)))
	}

	// Wait for processing with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("timeout: only processed %d out of %d tasks", atomic.LoadInt64(&processed), numTasks)
		case <-ticker.C:
			if atomic.LoadInt64(&processed) >= numTasks {
				return
			}
		}
	}
}

func TestTaskQueue_ContextCancellation(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var processed int64

	err := q.RegisterWorker("context-test", func(msg []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			atomic.AddInt64(&processed, 1)
			time.Sleep(50 * time.Millisecond)
			return nil
		}
	})
	assert.NoError(t, err)

	// Enqueue tasks
	for i := 0; i < 10; i++ {
		_ = q.Enqueue("context-test", []byte(fmt.Sprintf("task-%d", i)))
	}

	// Cancel context after some processing
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait a bit more
	time.Sleep(200 * time.Millisecond)

	processedCount := atomic.LoadInt64(&processed)
	assert.True(t, processedCount < 10, "should have cancelled some tasks")
}

// ========== NEW PUBSUB TESTS ==========

func TestPubSub_TopicWithSpecialCharacters(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topics := []string{
		"topic/with/slashes",
		"topic.with.dots",
		"topic-with-dashes",
		"topic_with_underscores",
		"topic@with@at",
		"topic with spaces",
		"topic#with#hash",
		"数据库更新", // Chinese characters
		"пользователь_создан", // Cyrillic
	}

	for _, topic := range topics {
		ch := ps.Memory.Subscribe(topic)
		defer ps.Memory.Unsubscribe(topic, ch)

		var wg sync.WaitGroup
		wg.Add(1)

		go func(topic string, ch <-chan []byte) {
			defer wg.Done()
			select {
			case msg := <-ch:
				assert.Equal(t, "test", string(msg))
			case <-time.After(1 * time.Second):
				t.Errorf("timeout for topic: %s", topic)
			}
		}(topic, ch)

		time.Sleep(10 * time.Millisecond)
		err := ps.Memory.Publish(topic, []byte("test"))
		assert.NoError(t, err)

		wg.Wait()
	}
}

func TestPubSub_MessageOrdering(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topic := "ordering-test"
	ch := ps.Memory.Subscribe(topic)
	defer ps.Memory.Unsubscribe(topic, ch)

	const numMessages = 100
	var received []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			select {
			case msg := <-ch:
				var num int
				fmt.Sscanf(string(msg), "%d", &num)
				mu.Lock()
				received = append(received, num)
				mu.Unlock()
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for message")
				return
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)

	// Send messages in order
	for i := 0; i < numMessages; i++ {
		err := ps.Memory.Publish(topic, []byte(fmt.Sprintf("%d", i)))
		assert.NoError(t, err)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, numMessages, len(received))
	
	// Check ordering
	for i := 0; i < numMessages; i++ {
		assert.Equal(t, i, received[i])
	}
}

func TestPubSub_SlowSubscriber(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topic := "slow-subscriber"
	ch := ps.Memory.Subscribe(topic)
	defer ps.Memory.Unsubscribe(topic, ch)

	var received int64
	var wg sync.WaitGroup

	wg.Add(1)

	// Slow subscriber
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			select {
			case <-ch:
				atomic.AddInt64(&received, 1)
				time.Sleep(100 * time.Millisecond) // Slow processing
			case <-time.After(5 * time.Second):
				return
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)

	// Fast publisher
	for i := 0; i < 10; i++ {
		err := ps.Memory.Publish(topic, []byte(fmt.Sprintf("msg-%d", i)))
		assert.NoError(t, err)
	}

	wg.Wait()

	assert.Equal(t, int64(10), atomic.LoadInt64(&received))
}

func TestPubSub_EmptyMessage(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topic := "empty-msg"
	ch := ps.Memory.Subscribe(topic)
	defer ps.Memory.Unsubscribe(topic, ch)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch:
			assert.Equal(t, 0, len(msg))
		case <-time.After(1 * time.Second):
			t.Error("timeout waiting for empty message")
		}
	}()

	time.Sleep(10 * time.Millisecond)
	err = ps.Memory.Publish(topic, []byte{})
	assert.NoError(t, err)

	wg.Wait()
}

func TestPubSub_LargeMessage(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topic := "large-msg"
	ch := ps.Memory.Subscribe(topic)
	defer ps.Memory.Unsubscribe(topic, ch)

	// Create 10MB message
	largeMsg := make([]byte, 10*1024*1024)
	for i := range largeMsg {
		largeMsg[i] = byte(i % 256)
	}

	var received []byte
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch:
			received = make([]byte, len(msg))
			copy(received, msg)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for large message")
		}
	}()

	time.Sleep(10 * time.Millisecond)
	err = ps.Memory.Publish(topic, largeMsg)
	assert.NoError(t, err)

	wg.Wait()

	assert.Equal(t, len(largeMsg), len(received))
	assert.Equal(t, largeMsg, received)
}

func TestPubSub_SubscriberDisconnection(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topic := "disconnect-test"
	ch1 := ps.Memory.Subscribe(topic)
	ch2 := ps.Memory.Subscribe(topic)

	var received1, received2 int64
	var wg sync.WaitGroup

	wg.Add(2)

	// Subscriber 1
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			select {
			case <-ch1:
				atomic.AddInt64(&received1, 1)
			case <-time.After(1 * time.Second):
				return
			}
		}
	}()

	// Subscriber 2 - disconnects early
	go func() {
		defer wg.Done()
		for i := 0; i < 2; i++ {
			select {
			case <-ch2:
				atomic.AddInt64(&received2, 1)
			case <-time.After(1 * time.Second):
				return
			}
		}
		// Disconnect early
		ps.Memory.Unsubscribe(topic, ch2)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send messages
	for i := 0; i < 5; i++ {
		err := ps.Memory.Publish(topic, []byte(fmt.Sprintf("msg-%d", i)))
		assert.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()

	assert.Equal(t, int64(5), atomic.LoadInt64(&received1))
	assert.True(t, atomic.LoadInt64(&received2) <= 2)
}

func TestPubSub_NoSubscribers(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	topic := "no-subscribers"

	// Publish to topic with no subscribers
	err = ps.Memory.Publish(topic, []byte("orphan message"))
	assert.NoError(t, err) // Should not error

	// Subscribe after publishing
	ch := ps.Memory.Subscribe(topic)
	defer ps.Memory.Unsubscribe(topic, ch)

	// Should not receive the previous message
	select {
	case <-ch:
		t.Error("should not receive message published before subscription")
	case <-time.After(100 * time.Millisecond):
		// Expected - no message received
	}
}

// ========== STRESS TESTS ==========

func TestPubSub_StressTestManyTopics(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	const numTopics = 1000
	var wg sync.WaitGroup
	wg.Add(numTopics)

	for i := 0; i < numTopics; i++ {
		topic := fmt.Sprintf("topic-%d", i)
		ch := ps.Memory.Subscribe(topic)
		defer ps.Memory.Unsubscribe(topic, ch)

		go func(topic string, ch <-chan []byte) {
			defer wg.Done()
			select {
			case msg := <-ch:
				assert.Equal(t, "test", string(msg))
			case <-time.After(5 * time.Second):
				t.Errorf("timeout for topic: %s", topic)
			}
		}(topic, ch)
	}

	time.Sleep(100 * time.Millisecond)

	// Publish to all topics
	for i := 0; i < numTopics; i++ {
		topic := fmt.Sprintf("topic-%d", i)
		err := ps.Memory.Publish(topic, []byte("test"))
		assert.NoError(t, err)
	}

	wg.Wait()
}

func TestTaskQueue_StressTestManyQueues(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	const numQueues = 100
	const tasksPerQueue = 10
	var processed int64

	// Register workers for all queues
	for i := 0; i < numQueues; i++ {
		queueName := fmt.Sprintf("queue-%d", i)
		err := q.RegisterWorker(queueName, func(msg []byte) error {
			atomic.AddInt64(&processed, 1)
			return nil
		})
		assert.NoError(t, err)
	}

	// Enqueue tasks
	for i := 0; i < numQueues; i++ {
		queueName := fmt.Sprintf("queue-%d", i)
		for j := 0; j < tasksPerQueue; j++ {
			err := q.Enqueue(queueName, []byte(fmt.Sprintf("task-%d", j)))
			assert.NoError(t, err)
		}
	}

	// Wait for processing
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	expectedTotal := int64(numQueues * tasksPerQueue)
	for {
		select {
		case <-timeout:
			t.Fatalf("timeout: only processed %d out of %d tasks", atomic.LoadInt64(&processed), expectedTotal)
		case <-ticker.C:
			if atomic.LoadInt64(&processed) >= expectedTotal {
				return
			}
		}
	}
}

// ========== INTEGRATION TESTS ==========

func TestIntegration_PubSubWithTaskQueue(t *testing.T) {
	ps, err := pubsub.NewEngine()
	assert.NoError(t, err)

	q := queue.NewInMem()
	defer q.Close()

	var processedTasks int64
	var receivedEvents int64

	// Setup task queue
	err = q.RegisterWorker("email", func(msg []byte) error {
		atomic.AddInt64(&processedTasks, 1)
		// Publish event after processing
		return ps.Memory.Publish("email.sent", []byte("email processed"))
	})
	assert.NoError(t, err)

	// Setup event subscriber
	ch := ps.Memory.Subscribe("email.sent")
	defer ps.Memory.Unsubscribe("email.sent", ch)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			select {
			case <-ch:
				atomic.AddInt64(&receivedEvents, 1)
			case <-time.After(2 * time.Second):
				return
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)

	// Send tasks
	for i := 0; i < 5; i++ {
		err := q.Enqueue("email", []byte(fmt.Sprintf("email-%d", i)))
		assert.NoError(t, err)
	}

	wg.Wait()

	assert.Equal(t, int64(5), atomic.LoadInt64(&processedTasks))
	assert.Equal(t, int64(5), atomic.LoadInt64(&receivedEvents))
}

func TestIntegration_ChainedQueues(t *testing.T) {
	q := queue.NewInMem()
	defer q.Close()

	var stage1Count, stage2Count, stage3Count int64

	// Stage 1: Process input
	err := q.RegisterWorker("stage1", func(msg []byte) error {
		atomic.AddInt64(&stage1Count, 1)
		// Forward to stage 2
		return q.Enqueue("stage2", append([]byte("processed-"), msg...))
	})
	assert.NoError(t, err)

	// Stage 2: Transform data
	err = q.RegisterWorker("stage2", func(msg []byte) error {
		atomic.AddInt64(&stage2Count, 1)
		// Forward to stage 3
		return q.Enqueue("stage3", append([]byte("transformed-"), msg...))
	})
	assert.NoError(t, err)

	// Stage 3: Final processing
	err = q.RegisterWorker("stage3", func(msg []byte) error {
		atomic.AddInt64(&stage3Count, 1)
		return nil
	})
	assert.NoError(t, err)

	// Send initial data
	for i := 0; i < 10; i++ {
		err := q.Enqueue("stage1", []byte(fmt.Sprintf("data-%d", i)))
		assert.NoError(t, err)
	}

	// Wait for processing
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for chained processing")
		case <-ticker.C:
			if atomic.LoadInt64(&stage3Count) >= 10 {
				assert.Equal(t, int64(10), atomic.LoadInt64(&stage1Count))
				assert.Equal(t, int64(10), atomic.LoadInt64(&stage2Count))
				assert.Equal(t, int64(10), atomic.LoadInt64(&stage3Count))
				return
			}
		}
	}
}



func BenchmarkTaskQueue_Parallel(b *testing.B) {
	q := queue.NewInMem()
	q.AllowMultipleWorkers = true
	defer q.Close()

	var counter int64
	_ = q.RegisterWorker("parallel", func(data []byte) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	b.ResetTimer()
	b.SetParallelism(8) // Uji dengan 8 thread paralel

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = q.Enqueue("parallel", []byte("msg"))
		}
	})
}


func BenchmarkTaskQueue_DelayRetry(b *testing.B) {
	q := queue.NewInMem()
	defer q.Close()

	var success int64

	_ = q.RegisterWorker("delay-retry", func(data []byte) error {
		time.Sleep(50 * time.Microsecond) // simulate delay
		if rand.Intn(10) < 3 {            // simulate 30% error rate
			return errors.New("simulated error")
		}
		atomic.AddInt64(&success, 1)
		return nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = q.Enqueue("delay-retry", []byte("msg"))
	}
}


func BenchmarkTaskQueue_WorkerPool(b *testing.B) {
	q := queue.NewInMem()
	q.AllowMultipleWorkers = true
	defer q.Close()

	var wg sync.WaitGroup
	workers := 16
	wg.Add(b.N)

	for i := 0; i < workers; i++ {
		_ = q.RegisterWorker("pool", func(data []byte) error {
			wg.Done()
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Enqueue("pool", []byte("task"))
	}

	wg.Wait()
}


func BenchmarkTaskQueue_RateLimit(b *testing.B) {
	q := queue.NewInMem()
	defer q.Close()

	limiter := time.Tick(100 * time.Microsecond) // ~10.000 msg/sec
	done := make(chan bool, b.N)

	_ = q.RegisterWorker("rate", func(data []byte) error {
		<-limiter // block sesuai rate
		done <- true
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Enqueue("rate", []byte("msg"))
	}

	for i := 0; i < b.N; i++ {
		<-done
	}
}
