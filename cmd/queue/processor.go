package main

import (
    "context"
    "errors"
    "log"
    "math/rand"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"
)

// simulateWork performs a fake task: 100–500ms, ~20% failure rate.
func simulateWork() error {
    d := time.Duration(100+rand.Intn(401)) * time.Millisecond
    time.Sleep(d)
    if rand.Intn(100) < 20 {
        return errors.New("simulated failure")
    }
    return nil
}

// backoffDuration calculates exponential backoff with jitter.
func backoffDuration(attempt int) time.Duration {
    if attempt < 1 {
        attempt = 1
    }
    if attempt > 7 {
        attempt = 7
    }
    base := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
    jitter := time.Duration(rand.Intn(200)) * time.Millisecond
    return base + jitter
}

// timeSleep is a tiny sleep helper (ms) used in workerLoop.
func timeSleep(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// processTask runs a task with retries, updating in-memory state and logging.
func (s *Server) processTask(t Task) {
    s.setState(t.ID, StateRunning)
    log.Printf("task start id=%s", t.ID)
    if err := simulateWork(); err != nil {
        if s.getRetry(t.ID) < t.MaxRetries {
            attempt := s.incRetry(t.ID)
            delay := backoffDuration(attempt)
            log.Printf("task fail id=%s attempt=%d delay=%s error=%v", t.ID, attempt, delay, err)
            time.AfterFunc(delay, func() {
                s.mu.Lock()
                if s.shuttingDown {
                    s.mu.Unlock()
                    s.setState(t.ID, StateFailed)
                    log.Printf("task dropped due to shutdown id=%s", t.ID)
                    return
                }
                s.states[t.ID] = StateQueued
                s.mu.Unlock()
                select {
                case s.jobs <- t:
                    log.Printf("task requeued id=%s attempt=%d", t.ID, attempt)
                default:
                    s.setState(t.ID, StateFailed)
                    log.Printf("task retry dropped (queue full) id=%s attempt=%d", t.ID, attempt)
                }
            })
            return
        }
        s.setState(t.ID, StateFailed)
        log.Printf("task failed permanently id=%s", t.ID)
        return
    }
    s.setState(t.ID, StateDone)
    log.Printf("task done id=%s", t.ID)
}

// getenvInt reads positive ints from env with default.
func getenvInt(key string, def int) int {
    v := os.Getenv(key)
    if v == "" {
        return def
    }
    n, err := strconv.Atoi(v)
    if err != nil || n <= 0 {
        return def
    }
    return n
}

// shutdown stops HTTP, then the pool, then marks remaining queued tasks failed.
func (s *Server) shutdown(ctx context.Context) error {
    var err error
    s.shutdownOnce.Do(func() {
        s.mu.Lock()
        s.shuttingDown = true
        s.mu.Unlock()

        log.Printf("shutdown: stopping http server")
        _ = s.httpServer.Shutdown(ctx)

        log.Printf("shutdown: stopping worker pool")
        s.pool.Stop()

        log.Printf("shutdown: marking remaining queued tasks as failed")
        for {
            select {
            case t := <-s.jobs:
                s.setState(t.ID, StateFailed)
                log.Printf("shutdown: failed queued id=%s", t.ID)
            default:
                log.Printf("shutdown: complete")
                return
            }
        }
    })
    return err
}

// run bootstraps the service and installs signal handling.
func run() {
    rand.Seed(time.Now().UnixNano())
    workers := getenvInt("WORKERS", 4)
    queueSize := getenvInt("QUEUE_SIZE", 64)
    srv := newServer(workers, queueSize)
    go func() {
        log.Printf("listening on :8080 (workers=%d, queue=%d)", workers, queueSize)
        if err := srv.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("http server error: %v", err)
        }
    }()

    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = srv.shutdown(ctx)
}


