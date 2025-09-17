package worker_pool

import (
	"errors"
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
			_ = wp.Submit(func() error {
				defer wg.Done()
				time.Sleep(5 * time.Millisecond)
				mu.Lock()
				completed[taskID] = true
				mu.Unlock()
				return nil
			})
		}

		wg.Wait()

		for i, done := range completed {
			if !done {
				t.Errorf("Задача %d не выполнена", i)
			}
		}
	})

	t.Run("SubmitWait должен ждать завершения задачи и вернуть ошибку", func(t *testing.T) {
		wp := NewWorkerPool(1)
		defer wp.Stop()

		start := time.Now()
		err := wp.SubmitWait(func() error {
			time.Sleep(100 * time.Millisecond)
			return errors.New("boom")
		})
		duration := time.Since(start)

		if duration < 90*time.Millisecond {
			t.Errorf("SubmitWait завершился слишком быстро: %v", duration)
		}
		if err == nil || err.Error() != "boom" {
			t.Errorf("ожидалась ошибка boom, получили: %v", err)
		}
	})

	t.Run("порядок выполнения задач FIFO", func(t *testing.T) {
		wp := NewWorkerPool(1) // один воркер -> последовательное выполнение
		defer wp.StopWait()

		var order []int
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			taskID := i
			_ = wp.Submit(func() error {
				defer wg.Done()
				mu.Lock()
				order = append(order, taskID)
				mu.Unlock()
				return nil
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
			_ = wp.Submit(func() error {
				time.Sleep(20 * time.Millisecond)
				mu.Lock()
				completed++
				mu.Unlock()
				return nil
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
		_ = wp.Submit(func() error {
			time.Sleep(100 * time.Millisecond)
			mu.Lock()
			completed++
			mu.Unlock()
			return nil
		})

		// эти задачи должны попасть в очередь
		for i := 0; i < 5; i++ {
			_ = wp.Submit(func() error {
				mu.Lock()
				completed++
				mu.Unlock()
				return nil
			})
		}

		// подождём, чтобы первая задача начала выполняться
		time.Sleep(10 * time.Millisecond)

		wp.Stop() // должен выполнить только первую, остальные отбросить

		if completed != 1 {
			t.Errorf("Stop должен был выполнить только текущую задачу, а выполнено %d", completed)
		}
	})

	t.Run("SubmitWait возвращает ошибку при панике задачи", func(t *testing.T) {
		wp := NewWorkerPool(1)
		defer wp.Stop()

		err := wp.SubmitWait(func() error {
			panic("panic inside task")
		})
		if err == nil {
			t.Fatalf("ожидалась ошибка из-за паники")
		}
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	wp := NewWorkerPool(4)
	defer wp.StopWait()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = wp.Submit(func() error {
				time.Sleep(time.Microsecond)
				return nil
			})
		}
	})
}

func BenchmarkSubmitWait(b *testing.B) {
	wp := NewWorkerPool(4)
	defer wp.StopWait()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wp.SubmitWait(func() error {
			time.Sleep(time.Microsecond)
			return nil
		})
	}
}
