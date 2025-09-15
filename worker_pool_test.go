package worker_pool

import (
	"sync"
	"testing"
	"time"
)

func TestWorkerPoolQueueBehavior(t *testing.T) {
	t.Run("все задачи должны выполниться", func(t *testing.T) {
		wp := NewWorkerPool(3)
		defer wp.StopWait()

		taskCount := 50
		completed := make([]bool, taskCount)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < taskCount; i++ {
			wg.Add(1)
			taskID := i
			wp.Submit(func() {
				defer wg.Done()
				time.Sleep(5 * time.Millisecond)
				mu.Lock()
				completed[taskID] = true
				mu.Unlock()
			})
		}

		wg.Wait()

		for i, done := range completed {
			if !done {
				t.Errorf("Задача %d не выполнена", i)
			}
		}
	})

	t.Run("SubmitWait должен ждать завершения задачи", func(t *testing.T) {
		wp := NewWorkerPool(1)
		defer wp.Stop()

		start := time.Now()
		wp.SubmitWait(func() {
			time.Sleep(100 * time.Millisecond)
		})
		duration := time.Since(start)

		if duration < 90*time.Millisecond {
			t.Errorf("SubmitWait завершился слишком быстро: %v", duration)
		}
	})

	t.Run("порядок выполнения задач FIFO", func(t *testing.T) {
		wp := NewWorkerPool(1) // один воркер → последовательное выполнение
		defer wp.StopWait()

		var order []int
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			taskID := i
			wp.Submit(func() {
				defer wg.Done()
				mu.Lock()
				order = append(order, taskID)
				mu.Unlock()
			})
		}

		wg.Wait()

		for i := 0; i < len(order); i++ {
			if order[i] != i {
				t.Errorf("Ожидался порядок %d, а получен %d (весь порядок: %v)", i, order[i], order)
				break
			}
		}
	})

	t.Run("StopWait должен дождаться всех задач в очереди", func(t *testing.T) {
		wp := NewWorkerPool(2)

		taskCount := 10
		var completed int
		var mu sync.Mutex

		for i := 0; i < taskCount; i++ {
			wp.Submit(func() {
				time.Sleep(20 * time.Millisecond)
				mu.Lock()
				completed++
				mu.Unlock()
			})
		}

		wp.StopWait()

		if completed != taskCount {
			t.Errorf("Ожидалось %d задач, выполнено %d", taskCount, completed)
		}
	})

	t.Run("Stop должен выполнить только текущие задачи и отбросить очередь", func(t *testing.T) {
		wp := NewWorkerPool(1)

		var completed int
		var mu sync.Mutex

		// первая задача гарантированно начнёт выполняться
		wp.Submit(func() {
			time.Sleep(100 * time.Millisecond)
			mu.Lock()
			completed++
			mu.Unlock()
		})

		// эти задачи должны попасть в очередь
		for i := 0; i < 5; i++ {
			wp.Submit(func() {
				mu.Lock()
				completed++
				mu.Unlock()
			})
		}

		// подождём, чтобы первая задача начала выполняться
		time.Sleep(10 * time.Millisecond)

		wp.Stop() // должен выполнить только первую, остальные отбросить

		if completed != 1 {
			t.Errorf("Stop должен был выполнить только текущую задачу, а выполнено %d", completed)
		}
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	wp := NewWorkerPool(4)
	defer wp.StopWait()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wp.Submit(func() {
				time.Sleep(time.Microsecond)
			})
		}
	})
}

func BenchmarkSubmitWait(b *testing.B) {
	wp := NewWorkerPool(4)
	defer wp.StopWait()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wp.SubmitWait(func() {
			time.Sleep(time.Microsecond)
		})
	}
}
