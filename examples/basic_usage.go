package main

import (
    "fmt"
    "sync"
    "time"
    "worker_pool"
)

func main() {
    // Создаем пул с 3 воркерами, размер очереди 100 и хук после каждой задачи
    wp := worker_pool.NewWorkerPool(3, 100, func() {
        // post-task hook (демо)
        fmt.Println("task done")
    })
    defer func() { _ = wp.Stop() }()

    var wg sync.WaitGroup
    totalTasks := 100

    fmt.Printf("Добавляем %d задач в очередь...\n", totalTasks)
    
    // Счетчик выполненных задач
    var completedTasks int
    var mu sync.Mutex

    // Добавляем задачи в очередь
    for i := 0; i < totalTasks; i++ {
        wg.Add(1)
        taskID := i
        if err := wp.Submit(func() {
            defer wg.Done()
            // Имитируем работу
            time.Sleep(100 * time.Millisecond)
            mu.Lock()
            completedTasks++
            current := completedTasks
            mu.Unlock()
            fmt.Printf("Задача %d завершена (всего: %d/%d)\n", taskID, current, totalTasks)
        }); err != nil {
            wg.Done()
            fmt.Printf("Submit error: %v\n", err)
        }
    }

    // Ждем завершения всех задач
    wg.Wait()
    fmt.Printf("\nВсе задачи завершены! Выполнено: %d из %d\n", completedTasks, totalTasks)

    // Демонстрация SubmitWait
    fmt.Println("\n=== Демонстрация SubmitWait ===")
    start := time.Now()
    _ = wp.SubmitWait(func() {
        time.Sleep(200 * time.Millisecond)
        fmt.Println("Задача с SubmitWait завершена!")
    })
    duration := time.Since(start)
    fmt.Printf("SubmitWait занял: %v\n", duration)

    // Демонстрация Stop (ожидает все задачи)
    fmt.Println("\n=== Демонстрация Stop ===")
    wp2 := worker_pool.NewWorkerPool(2, 10, nil)
    
    // Добавляем несколько задач
    for i := 0; i < 5; i++ {
        taskID := i
        _ = wp2.Submit(func() {
            time.Sleep(100 * time.Millisecond)
            fmt.Printf("Задача %d в Stop примере завершена\n", taskID)
        })
    }
    fmt.Println("Останавливаем пул со Stop...")
    _ = wp2.Stop()
    fmt.Println("Пул остановлен, все задачи выполнены!")
}