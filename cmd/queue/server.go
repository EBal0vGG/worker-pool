package main

import (
    "encoding/json"
    "log"
    "net/http"
    "sync"

    wpkg "worker_pool"
)

// Server wires HTTP endpoints to an internal buffered queue and a worker pool.
type Server struct {
    httpServer   *http.Server
    jobs         chan Task
    states       map[string]TaskState
    retries      map[string]int
    mu           sync.Mutex
    shuttingDown bool
    shutdownOnce sync.Once
    pool         *wpkg.WorkerPool
}

// newServer constructs a Server and starts queue readers.
func newServer(workers, queueSize int) *Server {
    s := &Server{
        jobs:    make(chan Task, queueSize),
        states:  make(map[string]TaskState, queueSize),
        retries: make(map[string]int, queueSize),
        pool:    wpkg.NewWorkerPool(workers),
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/enqueue", s.handleEnqueue)
    mux.HandleFunc("/healthz", s.handleHealth)
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        _, _ = w.Write([]byte("Worker Queue API\n\nPOST /enqueue {id,payload,max_retries}\nGET /healthz\n"))
    })
    s.httpServer = &http.Server{Addr: ":8080", Handler: mux}

    // Start queue readers; each reader submits jobs to the pool.
    for i := 0; i < workers; i++ {
        go s.workerLoop()
    }

    return s
}

// handleHealth returns 200 OK.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("ok"))
}

// handleEnqueue validates input and enqueues a task if buffer has space.
func (s *Server) handleEnqueue(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    s.mu.Lock()
    if s.shuttingDown {
        s.mu.Unlock()
        http.Error(w, "shutting down", http.StatusServiceUnavailable)
        return
    }
    s.mu.Unlock()

    var t Task
    dec := json.NewDecoder(r.Body)
    if err := dec.Decode(&t); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }
    if t.ID == "" {
        http.Error(w, "missing id", http.StatusBadRequest)
        return
    }
    if t.MaxRetries < 0 {
        http.Error(w, "max_retries must be >= 0", http.StatusBadRequest)
        return
    }

    // Mark as queued and try to place into the channel
    s.mu.Lock()
    if _, exists := s.states[t.ID]; !exists {
        s.states[t.ID] = StateQueued
        s.retries[t.ID] = 0
    }
    s.mu.Unlock()

    select {
    case s.jobs <- t:
        log.Printf("enqueue accepted id=%s max_retries=%d", t.ID, t.MaxRetries)
        w.WriteHeader(http.StatusAccepted)
        _, _ = w.Write([]byte("enqueued"))
    default:
        log.Printf("enqueue rejected (queue full) id=%s", t.ID)
        http.Error(w, "queue full", http.StatusServiceUnavailable)
    }
}

func (s *Server) setState(id string, st TaskState) {
    s.mu.Lock()
    s.states[id] = st
    s.mu.Unlock()
}

func (s *Server) incRetry(id string) int {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.retries[id] = s.retries[id] + 1
    return s.retries[id]
}

func (s *Server) getRetry(id string) int {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.retries[id]
}

func (s *Server) workerLoop() {
    stop := make(chan struct{})
    // Stop readers when the underlying pool stops
    go func() {
        for s.pool.IsRunning() {
            timeSleep(50)
        }
        close(stop)
    }()

    for {
        select {
        case <-stop:
            return
        case t := <-s.jobs:
            task := t
            _ = s.pool.Submit(func() error { s.processTask(task); return nil })
        }
    }
}


