package worker_pool

import (
    "context"
    "errors"
    "log"
    "runtime/debug"
    "sync"
)

type WorkerPool struct {
	workers   int
	taskQueue chan func()

	waitGroup sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewWorkerPool — создаёт пул воркеров
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
	}

	for i := 0; i < numberOfWorkers; i++ {
		wp.waitGroup.Add(1)
		go wp.worker()
	}

	return wp
}

// worker — воркер, выполняющий задачи
func (wp *WorkerPool) worker() {
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
                func() {
                    defer func() {
                        if r := recover(); r != nil {
                            log.Printf("worker recovered panic: %v\n%s", r, debug.Stack())
                        }
                    }()
                    task()
                }()
			}
		}
	}
}

// Submit — добавить задачу в пул
func (wp *WorkerPool) Submit(task func() error) error {
    if task == nil {
        return nil
    }

    wrapped := func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("task panic: %v\n%s", r, debug.Stack())
            }
        }()
        if err := task(); err != nil {
            log.Printf("task error: %v", err)
        }
    }

    select {
    case wp.taskQueue <- wrapped:
        return nil
    default:
        return errors.New("worker pool queue is full")
    }
}

// SubmitWait — добавить задачу и дождаться её завершения
func (wp *WorkerPool) SubmitWait(task func() error) error {
    if task == nil {
        return nil
    }

    done := make(chan error, 1)
    wrappedTask := func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("task panic: %v\n%s", r, debug.Stack())
                done <- errors.New("task panicked")
            }
        }()
        done <- task()
    }

    wp.taskQueue <- wrappedTask
    return <-done
}

// Stop — выполнить только текущие задачи, отбросив очередь
func (wp *WorkerPool) Stop() {
cleanup:
	for {
		select {
		case <-wp.taskQueue:
			// выбрасываем задачи
		default:
			break cleanup
		}
	}

	wp.cancel()
	wp.waitGroup.Wait()
}

// StopWait — дождаться выполнения всех задач в очереди
func (wp *WorkerPool) StopWait() {
	close(wp.taskQueue)
	wp.waitGroup.Wait()
}

// IsRunning — проверка, есть ли ещё активные воркеры
func (wp *WorkerPool) IsRunning() bool {
	select {
	case <-wp.ctx.Done():
		return false
	default:
		return true
	}
}
