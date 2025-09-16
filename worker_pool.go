package worker_pool

import (
	"errors"
	"sync"
)

type Pool interface {
	// Submit - добавить задачу в пул.
	// Если пул не имеет свободных воркеров, то задачу нужно добавить в очередь.
	// Если очередь переполнена, вернуть ошибку.
	Submit(task func()) error

	// Stop - остановить воркер пул, дождаться выполнения всех добавленных ранее в очередь задач.
	Stop() error
}

var _ Pool = (*WorkerPool)(nil)

type WorkerPool struct {
	workers   int
	taskQueue chan func()

	waitGroup sync.WaitGroup

	mu        sync.RWMutex
	isRunning bool

	afterTaskHook func()
}

func NewWorkerPool(numberOfWorkers int, queueSize int, afterTaskHook func()) *WorkerPool {
	if numberOfWorkers <= 0 {
		numberOfWorkers = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}

	wp := &WorkerPool{
		workers:       numberOfWorkers,
		taskQueue:     make(chan func(), queueSize),
		isRunning:     true,
		afterTaskHook: afterTaskHook,
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
	
	for task := range wp.taskQueue {
		if task == nil {
			continue
		}
		task()
		if wp.afterTaskHook != nil {
			wp.afterTaskHook()
		}
	}
}

// Submit — добавить задачу в пул. Если очередь переполнена, вернуть ошибку
func (wp *WorkerPool) Submit(task func()) error {
	if task == nil {
		return errors.New("task is nil")
	}

	wp.mu.RLock()
	running := wp.isRunning
	wp.mu.RUnlock()

	if !running {
		return errors.New("worker pool is stopped")
	}

	select {
	case wp.taskQueue <- task:
		return nil
	default:
		return errors.New("task queue is full")
	}
}

// SubmitWait — добавить задачу и дождаться её завершения
func (wp *WorkerPool) SubmitWait(task func()) error {
	if task == nil {
		return errors.New("task is nil")
	}

	done := make(chan struct{})

	wrappedTask := func() {
		task()
		close(done)
	}

	if err := wp.Submit(wrappedTask); err != nil {
		return err
	}
	<-done
	return nil
}

// Stop — остановить пул и дождаться выполнения всех задач из очереди
func (wp *WorkerPool) Stop() error {
	wp.mu.Lock()
	if !wp.isRunning {
		wp.mu.Unlock()
		return nil
	}
	wp.isRunning = false
	close(wp.taskQueue)
	wp.mu.Unlock()

	// Ждём завершения всех воркеров
	wp.waitGroup.Wait()
	return nil
}

// StopWait — совместимость со старым API, эквивалент Stop
func (wp *WorkerPool) StopWait() error { return wp.Stop() }

// IsRunning — вернуть статус пула
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.isRunning
}
