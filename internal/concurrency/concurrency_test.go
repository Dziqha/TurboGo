package concurrency

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAsync_ExecutesFunction(t *testing.T) {
	done := make(chan bool)
	Async(func() {
		done <- true
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Async did not execute within timeout")
	}
}

func TestAsync_PanicsDontLeak(t *testing.T) {
	Async(func() {
		panic("controlled panic")
	})

	time.Sleep(50 * time.Millisecond)
}

func TestWaitGroupRunner_AllComplete(t *testing.T) {
	var counter atomic.Int32
	funcs := make([]func(), 10)
	for i := range funcs {
		funcs[i] = func() {
			counter.Add(1)
		}
	}

	WaitGroupRunner(funcs...)
	assert.Equal(t, int32(10), counter.Load())
}

func TestWaitGroupRunner_Empty(t *testing.T) {
	WaitGroupRunner()
}

func TestWaitGroupRunner_ConcurrentExecution(t *testing.T) {
	var counter atomic.Int32
	start := make(chan struct{})

	funcs := make([]func(), 5)
	for i := range funcs {
		funcs[i] = func() {
			<-start
			counter.Add(1)
		}
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(start)
	}()

	WaitGroupRunner(funcs...)
	assert.Equal(t, int32(5), counter.Load())
}

func TestAsync_MultipleExecutions(t *testing.T) {
	var counter atomic.Int32
	for i := 0; i < 100; i++ {
		Async(func() {
			counter.Add(1)
		})
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(100), counter.Load())
}
