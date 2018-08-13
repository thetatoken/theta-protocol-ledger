package common

import "sync"

// Parallel executes multiple tasks in parallel
func Parallel(tasks ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(task func()) {
			task()
			wg.Done()
		}(task)
	}
	wg.Wait()
}
