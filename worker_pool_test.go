package worker_pool

import (
	"sync"
	"testing"
	"time"
)

func TestWorkerPoolQueueBehavior(t *testing.T) {
	t.Run("все задачи должны выполниться", func(t *testing.T) {
		wp := NewWorkerPool(3, 100, nil)
		defer wp.Stop()

		taskCount := 50
		completed := make([]bool, taskCount)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < taskCount; i++ {
			wg.Add(1)
			taskID := i
			if err := wp.Submit(func() {
				defer wg.Done()
				time.Sleep(5 * time.Millisecond)
				mu.Lock()
				completed[taskID] = true
				mu.Unlock()
			}); err != nil {
				t.Fatalf("Submit error: %v", err)
			}
		}

		wg.Wait()

		for i, done := range completed {
			if !done {
				t.Errorf("Задача %d не выполнена", i)
			}
		}
	})

	t.Run("SubmitWait должен ждать завершения задачи", func(t *testing.T) {
		wp := NewWorkerPool(1, 10, nil)
		defer wp.Stop()

		start := time.Now()
		if err := wp.SubmitWait(func() {
			time.Sleep(100 * time.Millisecond)
		}); err != nil {
			t.Fatalf("SubmitWait error: %v", err)
		}
		duration := time.Since(start)

		if duration < 90*time.Millisecond {
			t.Errorf("SubmitWait завершился слишком быстро: %v", duration)
		}
	})

	t.Run("порядок выполнения задач FIFO", func(t *testing.T) {
		wp := NewWorkerPool(1, 100, nil) // один воркер → последовательное выполнение
		defer wp.Stop()

		var order []int
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			taskID := i
			if err := wp.Submit(func() {
				defer wg.Done()
				mu.Lock()
				order = append(order, taskID)
				mu.Unlock()
			}); err != nil {
				t.Fatalf("Submit error: %v", err)
			}
		}

		wg.Wait()

		for i := 0; i < len(order); i++ {
			if order[i] != i {
				t.Errorf("Ожидался порядок %d, а получен %d (весь порядок: %v)", i, order[i], order)
				break
			}
		}
	})

	t.Run("Stop должен дождаться всех задач в очереди", func(t *testing.T) {
		wp := NewWorkerPool(2, 100, nil)

		taskCount := 10
		var completed int
		var mu sync.Mutex

		for i := 0; i < taskCount; i++ {
			if err := wp.Submit(func() {
				time.Sleep(20 * time.Millisecond)
				mu.Lock()
				completed++
				mu.Unlock()
			}); err != nil {
				t.Fatalf("Submit error: %v", err)
			}
		}

		if err := wp.Stop(); err != nil {
			t.Fatalf("Stop error: %v", err)
		}

		if completed != taskCount {
			t.Errorf("Ожидалось %d задач, выполнено %d", taskCount, completed)
		}
	})

	t.Run("Stop/Submit при переполнении и остановке", func(t *testing.T) {
		wp := NewWorkerPool(1, 1, nil)

		started := make(chan struct{})
		// первая задача сигналит, что она началась (значит снята из очереди)
		if err := wp.Submit(func() {
			close(started)
			time.Sleep(50 * time.Millisecond)
		}); err != nil {
			t.Fatalf("Submit error: %v", err)
		}

		// Ждем, пока первая задача начнется
		<-started

		// эта задача должна занять единственный слот в очереди
		if err := wp.Submit(func() {}); err != nil {
			t.Fatalf("Submit error: %v", err)
		}

		// следующая должна вернуть ошибку переполнения
		if err := wp.Submit(func() {}); err == nil {
			t.Fatalf("ожидалась ошибка переполнения очереди")
		}

		if err := wp.Stop(); err != nil {
			t.Fatalf("Stop error: %v", err)
		}
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	wp := NewWorkerPool(4, 100000, nil)
	defer wp.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = wp.Submit(func() {
				time.Sleep(time.Microsecond)
			})
		}
	})
}

func BenchmarkSubmitWait(b *testing.B) {
	wp := NewWorkerPool(4, 100000, nil)
	defer wp.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wp.SubmitWait(func() {
			time.Sleep(time.Microsecond)
		})
	}
}
