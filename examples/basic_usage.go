package main

import (
    "fmt"
    "sync"
    "time"
    "worker_pool"
)

func main() {
    // Создаем пул с 3 воркерами
    wp := worker_pool.NewWorkerPool(3)
    defer wp.Stop()

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
        
        _ = wp.Submit(func() error {
            defer wg.Done()
            
            // Имитируем работу
            time.Sleep(100 * time.Millisecond)
            
            mu.Lock()
            completedTasks++
            current := completedTasks
            mu.Unlock()
            
            fmt.Printf("Задача %d завершена (всего: %d/%d)\n", taskID, current, totalTasks)
            return nil
        })
    }

    // Ждем завершения всех задач
    wg.Wait()
    fmt.Printf("\nВсе задачи завершены! Выполнено: %d из %d\n", completedTasks, totalTasks)

    // Демонстрация SubmitWait
    fmt.Println("\n=== Демонстрация SubmitWait ===")
    start := time.Now()
    _ = wp.SubmitWait(func() error {
        time.Sleep(200 * time.Millisecond)
        fmt.Println("Задача с SubmitWait завершена!")
        return nil
    })
    duration := time.Since(start)
    fmt.Printf("SubmitWait занял: %v\n", duration)

    // Демонстрация StopWait
    fmt.Println("\n=== Демонстрация StopWait ===")
    wp2 := worker_pool.NewWorkerPool(2)
    
    // Добавляем несколько задач
    for i := 0; i < 5; i++ {
        taskID := i
        _ = wp2.Submit(func() error {
            time.Sleep(100 * time.Millisecond)
            fmt.Printf("Задача %d в StopWait примере завершена\n", taskID)
            return nil
        })
    }
    
    fmt.Println("Останавливаем пул с StopWait...")
    wp2.StopWait()
    fmt.Println("Пул остановлен, все задачи выполнены!")
}