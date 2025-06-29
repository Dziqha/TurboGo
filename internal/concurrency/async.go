package concurrency

import "sync"

// RunAsync runs a function asynchronously in a goroutine and tracks its completion
func RunAsync(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}

// RunParallel runs multiple functions in parallel and waits for all to finish
func RunParallel(fns ...func()) {
	var wg sync.WaitGroup
	for _, fn := range fns {
		RunAsync(&wg, fn)
	}
	wg.Wait()
}
