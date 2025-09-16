# WorkerPool

Реализация пула воркеров на Go для эффективного управления горутинами и выполнения задач.

## Описание

WorkerPool предоставляет простой и эффективный способ управления горутинами для выполнения задач. Пул создает фиксированное количество воркеров, которые обрабатывают задачи из общей очереди.

## Возможности

- Создание пула с заданным количеством воркеров
- Асинхронное добавление задач через `Submit()`
- Синхронное добавление задач через `SubmitWait()`
- Гибкая остановка: `Stop()` и `StopWait()`
- Thread-safe операции
- Буферизованная очередь задач

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
    wp.Submit(func() {
        fmt.Println("Task 1 executed")
        time.Sleep(100 * time.Millisecond)
    })

    // Синхронное добавление задач
    wp.SubmitWait(func() {
        fmt.Println("Task 2 executed")
        time.Sleep(100 * time.Millisecond)
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
    wp.Submit(func() {
        defer wg.Done()
        // Выполнение задачи
        processTask(taskID)
    })
}
wg.Wait()
```

## API

### NewWorkerPool(numberOfWorkers int) *WorkerPool

Создает новый пул воркеров с указанным количеством воркеров.

**Параметры:**
- `numberOfWorkers` - количество воркеров (минимум 1)

### Submit(task func())

Добавляет задачу в очередь и возвращает управление немедленно. Если очередь переполнена, задача игнорируется.

**Параметры:**
- `task` - функция для выполнения

### SubmitWait(task func())

Добавляет задачу в очередь и блокирует выполнение до завершения задачи.

**Параметры:**
- `task` - функция для выполнения

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
    wp.Submit(func() {
        processRequest(request)
    })
}
```

### Параллельная обработка данных

```go
wp := worker_pool.NewWorkerPool(4)
defer wp.StopWait()

for _, data := range dataSlice {
    wp.SubmitWait(func() {
        result := processData(data)
        results = append(results, result)
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
    wp.Submit(func() {
        defer wg.Done()
        processBatch(batch)
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
├── examples/
│   └── basic_usage.go         # Примеры использования
├── go.mod                     # Go модуль
└── README.md                  # Документация
```

## Лицензия

MIT License
