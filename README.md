# WorkerPool

Реализация пула воркеров на Go для эффективного управления горутинами и выполнения задач с поддержкой ограниченной очереди и хуков.

## Описание

WorkerPool предоставляет простой и эффективный способ управления горутинами для выполнения задач. Пул создает фиксированное количество воркеров, которые обрабатывают задачи из общей ограниченной очереди.

## Возможности

- Создание пула с заданным количеством воркеров и размером очереди
- Асинхронное добавление задач через `Submit()` с обработкой ошибок
- Синхронное добавление задач через `SubmitWait()` с обработкой ошибок
- Остановка с ожиданием всех задач через `Stop()`
- Thread-safe операции
- Ограниченная очередь задач с ошибкой при переполнении
- Хук после выполнения каждой задачи
- Защита от паники в задачах и хуках
- Интерфейс `Pool` для типизации

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
    // Создаем пул с 3 воркерами, размер очереди 100, без хука
    wp := worker_pool.NewWorkerPool(3, 100, nil)
    defer wp.Stop()

    // Асинхронное добавление задач с обработкой ошибок
    if err := wp.Submit(func() {
        fmt.Println("Task 1 executed")
        time.Sleep(100 * time.Millisecond)
    }); err != nil {
        fmt.Printf("Submit error: %v\n", err)
    }

    // Синхронное добавление задач с обработкой ошибок
    if err := wp.SubmitWait(func() {
        fmt.Println("Task 2 executed")
        time.Sleep(100 * time.Millisecond)
    }); err != nil {
        fmt.Printf("SubmitWait error: %v\n", err)
    }
}
```

### Пример с хуком

```go
// Создаем пул с хуком после каждой задачи
wp := worker_pool.NewWorkerPool(4, 50, func() {
    fmt.Println("Task completed!")
})

defer wp.Stop()

for i := 0; i < 10; i++ {
    taskID := i
    if err := wp.Submit(func() {
        fmt.Printf("Executing task %d\n", taskID)
        time.Sleep(50 * time.Millisecond)
    }); err != nil {
        fmt.Printf("Submit error: %v\n", err)
    }
}
```

### Обработка множественных задач

```go
wp := worker_pool.NewWorkerPool(4, 100, nil)
defer wp.Stop()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    taskID := i
    if err := wp.Submit(func() {
        defer wg.Done()
        // Выполнение задачи
        processTask(taskID)
    }); err != nil {
        wg.Done()
        fmt.Printf("Submit error: %v\n", err)
    }
}
wg.Wait()
```

### Обработка переполнения очереди

```go
// Создаем пул с маленькой очередью для демонстрации
wp := worker_pool.NewWorkerPool(1, 2, nil)
defer wp.Stop()

// Добавляем задачи до переполнения
for i := 0; i < 5; i++ {
    taskID := i
    if err := wp.Submit(func() {
        fmt.Printf("Task %d executed\n", taskID)
        time.Sleep(100 * time.Millisecond)
    }); err != nil {
        fmt.Printf("Task %d failed: %v\n", taskID, err)
    }
}
```

### Защита от паники

```go
wp := worker_pool.NewWorkerPool(2, 10, func() {
    // Хук тоже защищен от паники
    panic("hook panic") // Будет перехвачено и выведено в лог
})

defer wp.Stop()

// Задача с паникой будет перехвачена
if err := wp.Submit(func() {
    panic("task panic") // Будет перехвачено и выведено в лог
}); err != nil {
    fmt.Printf("Submit error: %v\n", err)
}
```

## API

### Интерфейс Pool

```go
type Pool interface {
    // Submit - добавить задачу в пул.
    // Если очередь переполнена, вернуть ошибку.
    Submit(task func()) error

    // Stop - остановить воркер пул, дождаться выполнения всех добавленных ранее в очередь задач.
    Stop() error
}
```

### NewWorkerPool(numberOfWorkers int, queueSize int, afterTaskHook func()) *WorkerPool

Создает новый пул воркеров с указанным количеством воркеров, размером очереди и хуком.

**Параметры:**
- `numberOfWorkers` - количество воркеров (минимум 1)
- `queueSize` - размер очереди задач (минимум 1)
- `afterTaskHook` - функция, вызываемая после каждой задачи (может быть nil)

### Submit(task func()) error

Добавляет задачу в очередь и возвращает управление немедленно. Возвращает ошибку если:
- Задача равна nil
- Очередь переполнена

**Параметры:**
- `task` - функция для выполнения

**Возвращает:**
- `error` - ошибка при добавлении задачи

### SubmitWait(task func()) error

Добавляет задачу в очередь и блокирует выполнение до завершения задачи.

**Параметры:**
- `task` - функция для выполнения

**Возвращает:**
- `error` - ошибка при добавлении задачи

### Stop() error

Останавливает пул и ждет завершения всех задач, включая находящиеся в очереди. Закрывает канал задач, после чего воркеры завершаются естественным образом.

**Возвращает:**
- `error` - ошибка при остановке (в текущей реализации всегда nil)

### IsRunning() bool

Проверяет, работает ли пул. В текущей реализации всегда возвращает `true` для открытого канала.

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
wp := worker_pool.NewWorkerPool(10, 1000, func() {
    // Логирование завершения запроса
    log.Println("Request processed")
})
defer wp.Stop()

for request := range requests {
    if err := wp.Submit(func() {
        processRequest(request)
    }); err != nil {
        log.Printf("Failed to submit request: %v", err)
    }
}
```

### Параллельная обработка данных

```go
wp := worker_pool.NewWorkerPool(4, 100, nil)
defer wp.Stop()

for _, data := range dataSlice {
    if err := wp.SubmitWait(func() {
        result := processData(data)
        results = append(results, result)
    }); err != nil {
        log.Printf("Failed to process data: %v", err)
    }
}
```

### Batch обработка с хуком

```go
var processedCount int
var mu sync.Mutex

wp := worker_pool.NewWorkerPool(5, 200, func() {
    mu.Lock()
    processedCount++
    mu.Unlock()
})
defer wp.Stop()

var wg sync.WaitGroup
for _, batch := range batches {
    wg.Add(1)
    if err := wp.Submit(func() {
        defer wg.Done()
        processBatch(batch)
    }); err != nil {
        wg.Done()
        log.Printf("Failed to submit batch: %v", err)
    }
}
wg.Wait()
```

## Особенности реализации

- **Ограниченная очередь**: Очередь задач с настраиваемым размером
- **Обработка ошибок**: Все методы возвращают ошибки для лучшего контроля
- **Хук после задач**: Возможность выполнения кода после каждой задачи
- **Защита от паники**: Автоматический перехват паники в задачах и хуках
- **Thread-safe операции**: Безопасная работа с горутинами
- **Graceful shutdown**: Корректное завершение работы воркеров через закрытие канала
- **FIFO порядок**: Задачи выполняются в порядке поступления
- **Интерфейс Pool**: Типизация для лучшей архитектуры

## Производительность

WorkerPool оптимизирован для:
- Минимального overhead при создании задач
- Эффективного распределения нагрузки между воркерами
- Быстрой остановки и очистки ресурсов
- Контроля нагрузки через ограничение очереди
- Устойчивости к панике в пользовательском коде

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