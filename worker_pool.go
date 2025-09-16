package worker_pool

import (
	"context"
	"sync"
)

type WorkerPool struct {
	workers   int
	taskQueue chan func()

	waitGroup sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc

	mu        sync.RWMutex
	isRunning bool
}

func NewWorkerPool(numberOfWorkers int) *WorkerPool {
	if numberOfWorkers <= 0 {
		numberOfWorkers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		workers:   numberOfWorkers,
		taskQueue: make(chan func(), 100),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: true,
	}

	// Запускаем воркеров
	for i := 0; i < numberOfWorkers; i++ {
		wp.waitGroup.Add(1)
		go wp.worker(i)
	}

	return wp
}

// worker — горутина, которая обрабатывает задачи из очереди
func (wp *WorkerPool) worker(id int) {
	defer wp.waitGroup.Done()
	
	for {
		select {
		case <-wp.ctx.Done():
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			if task != nil {
				task()
			}
		}
	}	
}

// Submit — добавить задачу в пул
func (wp *WorkerPool) Submit(task func()) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.isRunning {
		return
	}

	select {
	case wp.taskQueue <- task:
		// Задача добавлена
	default:
		// Очередь переполнена — игнорируем
	}
}

// SubmitWait — добавить задачу и дождаться её завершения
func (wp *WorkerPool) SubmitWait(task func()) {
	wp.mu.RLock()
	if !wp.isRunning {
		wp.mu.RUnlock()
		return
	}

	done := make(chan struct{})

	wrappedTask := func() {
		task()
		close(done)
	}

	wp.taskQueue <- wrappedTask

	wp.mu.RUnlock()
	<-done
}

// Stop — остановить пул, выполнить только текущие задачи, отбросить очередь
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	if !wp.isRunning {
		wp.mu.Unlock()
		return
	}
	wp.isRunning = false
	wp.mu.Unlock()

	// Сигналим воркерам прекратить брать новые задачи
	wp.cancel()

	// Ждём выполнения только активных
	wp.waitGroup.Wait()
}

// StopWait — остановить пул и дождаться выполнения всех задач
func (wp *WorkerPool) StopWait() {
	wp.mu.Lock()
	if !wp.isRunning {
		wp.mu.Unlock()
		return
	}
	wp.isRunning = false
	wp.mu.Unlock()

	// Закрываем очередь задач
	close(wp.taskQueue)

	// Ждём завершения всех воркеров
	wp.waitGroup.Wait()
}

// IsRunning — вернуть статус пула
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.isRunning
}
