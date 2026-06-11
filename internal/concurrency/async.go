package concurrency

import "sync"

func Async(fn func()) {
	go func() {
		defer func() {
			recover()
		}()
		fn()
	}()
}

func WaitGroupRunner(funcs ...func()) {
	var wg sync.WaitGroup
	for _, fn := range funcs {
		wg.Add(1)
		go func(f func()) {
			defer wg.Done()
			f()
		}(fn)
	}
	wg.Wait()
}
