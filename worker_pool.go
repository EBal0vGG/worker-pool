package worker_pool

import (
	"errors"
	"sync"
	"fmt"
)

type Pool interface {
	Submit(task func()) error
	Stop() error
}

var _ Pool = (*WorkerPool)(nil)

type WorkerPool struct {
	workers   int
	taskQueue chan func()

	waitGroup sync.WaitGroup

	afterTaskHook func()
}

// NewWorkerPool — создаёт пул воркеров
func NewWorkerPool(numberOfWorkers, queueSize int, afterTaskHook func()) *WorkerPool {
	if numberOfWorkers <= 0 {
		numberOfWorkers = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}

	wp := &WorkerPool{
		workers:       numberOfWorkers,
		taskQueue:     make(chan func(), queueSize),
		afterTaskHook: afterTaskHook,
	}

	// Запускаем воркеров
	for i := 0; i < numberOfWorkers; i++ {
		wp.waitGroup.Add(1)
		go wp.worker(i)
	}

	return wp
}

// worker — воркер, обрабатывающий задачи
func (wp *WorkerPool) worker(id int) {
	defer wp.waitGroup.Done()

	// range по закрытому каналу завершается, когда канал пустой
	for task := range wp.taskQueue {
		if task == nil {
			continue
		}

		// Выполнение задачи с защитой от паники
		func() {
			defer func() { 
				r := recover()
				if r != nil {
					fmt.Printf("worker %d: panic in task: %v\n", id, r)
				}
			}()
			task()
		}()

		// Выполнение afterTaskHook
		if wp.afterTaskHook != nil {
			func() { 
				defer func() { 
					r := recover()
					if r != nil {
						fmt.Printf("worker %d: panic in afterTaskHook: %v\n", id, r)
					}
				}()
				wp.afterTaskHook() 
			}()
		}
	}
}

// Submit — добавить задачу в пул
func (wp *WorkerPool) Submit(task func()) error {
	if task == nil {
		return errors.New("task is nil")
	}

	select {
	case wp.taskQueue <- task:
		return nil
	default:
		return errors.New("task queue is full")
	}
}

// SubmitWait — добавить задачу и дождаться выполнения
func (wp *WorkerPool) SubmitWait(task func()) error {
	if task == nil {
		return errors.New("task is nil")
	}

	done := make(chan struct{})

	wrappedTask := func() {
		defer close(done)
		task()
	}

	select {
	case wp.taskQueue <- wrappedTask:
		<-done
		return nil
	default:
		return errors.New("task queue is full")
	}
}

// Stop — закрывает пул и ждёт выполнения всех задач
func (wp *WorkerPool) Stop() error {
	// Закрываем канал, воркеры продолжат выполнять задачи до конца очереди
	close(wp.taskQueue)

	// Ждём завершения всех воркеров
	wp.waitGroup.Wait()
	return nil
}

// IsRunning — проверка, есть ли ещё воркеры
func (wp *WorkerPool) IsRunning() bool {
	// Если канал закрыт — пул остановлен
	select {
	case <-wp.taskQueue:
		return true
	default:
		return true // канал открыт, значит пул работает
	}
}
