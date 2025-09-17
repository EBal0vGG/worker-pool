# WorkerPool

Реализация пула воркеров на Go и демонстрационного HTTP-сервиса очереди.

## Описание

WorkerPool предоставляет простой и эффективный способ управления горутинами для выполнения задач. Пул создает фиксированное количество воркеров, которые обрабатывают задачи из общей очереди.

## Возможности

- Создание пула с заданным количеством воркеров
- Асинхронное добавление задач через `Submit()`
- Синхронное добавление задач через `SubmitWait()`
- Гибкая остановка: `Stop()` и `StopWait()`
- Thread-safe операции
- Буферизованная очередь задач
- Демонстрационный HTTP-сервис очереди: приём задач, обработка пулом, ретраи

## Queue Server (demo)

Мини-сервис, который использует `WorkerPool` для фоновой обработки задач.

- Конфигурация (env):
  - `WORKERS` — число воркеров (по умолчанию 4)
  - `QUEUE_SIZE` — размер буферизированной очереди (по умолчанию 64)

- Запуск:
```bash
go run ./worker_pool/cmd/queue
```

- Эндпоинты:
  - `GET /healthz` -> 200 OK
  - `POST /enqueue` — тело JSON:
    ```json
    {"id":"<string>","payload":"<string>","max_retries":<int>}
    ```
    Ответ 202 (принято) или 503 (очередь переполнена).

- Поведение обработки:
  - Каждая задача «работает» 100–500 мс (симулируется)
  - ~20% задач завершаются с ошибкой (симулируется)
  - Экспоненциальный бэкофф с джиттером до `max_retries` попыток
  - Состояния задач (in-memory): `queued | running | done | failed`
  - Грейсфул-шатдаун по SIGINT/SIGTERM: перестаём принимать новые, ждём текущие

## Установка

```bash
go mod init worker_pool
```

## Использование

### Базовый пример

```go
package main

import (
    "fmt"
    "time"
    "worker_pool"
)

func main() {
    // Создаем пул с 3 воркерами
    wp := worker_pool.NewWorkerPool(3)
    defer wp.StopWait()

    // Асинхронное добавление задач
    _ = wp.Submit(func() error {
        fmt.Println("Task 1 executed")
        time.Sleep(100 * time.Millisecond)
        return nil
    })

    // Синхронное добавление задач
    _ = wp.SubmitWait(func() error {
        fmt.Println("Task 2 executed")
        time.Sleep(100 * time.Millisecond)
        return nil
    })
}
```

### Обработка множественных задач

```go
wp := worker_pool.NewWorkerPool(4)
defer wp.StopWait()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    taskID := i
    _ = wp.Submit(func() error {
        defer wg.Done()
        // Выполнение задачи
        return processTask(taskID)
    })
}
wg.Wait()
```

## API

### NewWorkerPool(numberOfWorkers int) *WorkerPool

Создает новый пул воркеров с указанным количеством воркеров.

**Параметры:**
- `numberOfWorkers` - количество воркеров (минимум 1)

### Submit(task func() error) error

Добавляет задачу в очередь и возвращает управление немедленно. Возвращает ошибку, если очередь переполнена.

**Параметры:**
- `task` - функция для выполнения, возвращающая ошибку

**Возвращает:**
- `error` - ошибка при переполнении очереди или `nil`

### SubmitWait(task func() error) error

Добавляет задачу в очередь и блокирует выполнение до завершения задачи. Возвращает ошибку задачи или панику.

**Параметры:**
- `task` - функция для выполнения, возвращающая ошибку

**Возвращает:**
- `error` - ошибка задачи, паника конвертируется в ошибку

### Stop()

Останавливает пул и ждет завершения только выполняющихся в данный момент задач. Задачи в очереди отбрасываются.

### StopWait()

Останавливает пул и ждет завершения всех задач, включая находящиеся в очереди.

### IsRunning() bool

Возвращает `true`, если пул активен.

## Тестирование

Запуск тестов:

```bash
go test -v
```

Запуск бенчмарков:

```bash
go test -bench=.
```

## Примеры использования

### Обработка HTTP запросов

```go
wp := worker_pool.NewWorkerPool(10)
defer wp.StopWait()

for request := range requests {
    _ = wp.Submit(func() error {
        return processRequest(request)
    })
}
```

### Параллельная обработка данных

```go
wp := worker_pool.NewWorkerPool(4)
defer wp.StopWait()

for _, data := range dataSlice {
    _ = wp.SubmitWait(func() error {
        result := processData(data)
        results = append(results, result)
        return nil
    })
}
```

### Batch обработка

```go
wp := worker_pool.NewWorkerPool(5)
defer wp.StopWait()

var wg sync.WaitGroup
for _, batch := range batches {
    wg.Add(1)
    _ = wp.Submit(func() error {
        defer wg.Done()
        return processBatch(batch)
    })
}
wg.Wait()
```

## Особенности реализации

- **Context-based cancellation**: Использует `context.Context` для корректной остановки
- **Buffered channel**: Очередь задач с буфером размером 100
- **Mutex protection**: Thread-safe операции с состоянием пула
- **Graceful shutdown**: Корректное завершение работы воркеров
- **FIFO порядок**: Задачи выполняются в порядке поступления
- **Panic recovery**: Паники в задачах логируются со стеком, воркеры не падают
- **Error handling**: Методы возвращают ошибки для обработки сбоев

## Производительность

WorkerPool оптимизирован для:
- Минимального overhead при создании задач
- Эффективного распределения нагрузки между воркерами
- Быстрой остановки и очистки ресурсов

## Структура проекта

```
worker_pool/
├── worker_pool.go             # Основная реализация
├── worker_pool_test.go        # Unit тесты
├── cmd/
│   └── queue/                 # HTTP-сервис очереди
│       ├── main.go            # Точка входа
│       ├── types.go           # Типы данных
│       ├── server.go          # HTTP-сервер и обработчики
│       └── processor.go       # Обработка задач и graceful shutdown
├── examples/
│   └── basic_usage.go         # Примеры использования
├── go.mod                     # Go модуль
└── README.md                  # Документация
```

## Лицензия

MIT License