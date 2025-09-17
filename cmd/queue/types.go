package main

// Task represents an incoming unit of work.
// Payload is opaque in this demo; only ID and retry config are used.
type Task struct {
    ID         string `json:"id"`
    Payload    string `json:"payload"`
    MaxRetries int    `json:"max_retries"`
}

// TaskState is an in-memory processing state for a task.
type TaskState string

const (
    StateQueued  TaskState = "queued"
    StateRunning TaskState = "running"
    StateDone    TaskState = "done"
    StateFailed  TaskState = "failed"
)


