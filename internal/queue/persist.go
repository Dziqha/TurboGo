package queue

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Task struct {
	Queue     string    `json:"queue"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
	Status    string    `json:"status"` // pending, processing, completed, failed
	Retries   int       `json:"retries"`
}

type PersistentTaskQueue struct {
	*TaskQueue
	logFile    string
	file       *os.File
	writeMutex sync.Mutex
	taskID     uint64
	idMutex    sync.Mutex
	tasks      map[string]*Task
	tasksMutex sync.RWMutex
}

func NewPersistent(logFile string) (*PersistentTaskQueue, error) {
	inMem := NewInMem()
	
	ptq := &PersistentTaskQueue{
		TaskQueue: inMem,
		logFile:   logFile,
		taskID:    1,
		tasks:     make(map[string]*Task),
	}
	
	if err := ptq.loadPendingTasks(); err != nil {
		return nil, fmt.Errorf("failed to load pending tasks: %v", err)
	}

	ptq.AutoCleanup(10*time.Minute, 24*time.Hour)
	
	return ptq, nil
}

func (ptq *PersistentTaskQueue) ensureFile() error {
	if ptq.file != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(ptq.logFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	file, err := os.OpenFile(ptq.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	ptq.file = file
	return nil
}

func (ptq *PersistentTaskQueue) Enqueue(queue string, task []byte) error {
	if err := ptq.ensureFile(); err != nil {
		return err
	}
	ptq.idMutex.Lock()
	id := fmt.Sprintf("%d", ptq.taskID)
	ptq.taskID++
	ptq.idMutex.Unlock()
	
	taskObj := &Task{
		Queue:     queue,
		Data:      task,
		Timestamp: time.Now(),
		ID:        id,
		Status:    "pending",
		Retries:   0,
	}
	
	// Store task
	ptq.tasksMutex.Lock()
	ptq.tasks[id] = taskObj
	ptq.tasksMutex.Unlock()
	
	if err := ptq.logTask(taskObj); err != nil {
		return fmt.Errorf("failed to log task: %v", err)
	}
	
	// Enqueue in-memory
	return ptq.TaskQueue.Enqueue(queue, task)
}

func (ptq *PersistentTaskQueue) RegisterWorker(queue string, handler func([]byte) error) error {
	// Wrap handler with persistence logic
	wrappedHandler := func(data []byte) error {
		// Find task by data (simplified approach)
		var taskID string
		ptq.tasksMutex.RLock()
		for id, task := range ptq.tasks {
			if string(task.Data) == string(data) && task.Status == "pending" {
				taskID = id
				break
			}
		}
		ptq.tasksMutex.RUnlock()
		
		if taskID != "" {
			ptq.updateTaskStatus(taskID, "processing")
		}
		
		// Execute handler
		err := handler(data)
		
		if taskID != "" {
			if err != nil {
				ptq.updateTaskStatus(taskID, "failed")
				ptq.incrementRetries(taskID)
			} else {
				ptq.updateTaskStatus(taskID, "completed")
			}
		}
		
		return err
	}
	
	return ptq.TaskQueue.RegisterWorker(queue, wrappedHandler)
}

func (ptq *PersistentTaskQueue) logTask(task *Task) error {
	ptq.writeMutex.Lock()
	defer ptq.writeMutex.Unlock()
	
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	
	_, err = ptq.file.Write(append(data, '\n'))
	if err != nil {
		return err
	}
	
	return ptq.file.Sync()
}

func (ptq *PersistentTaskQueue) updateTaskStatus(taskID, status string) {
	ptq.tasksMutex.Lock()
	defer ptq.tasksMutex.Unlock()
	
	if task, ok := ptq.tasks[taskID]; ok {
		task.Status = status
		ptq.logTask(task) // Re-log with updated status
	}
}

func (ptq *PersistentTaskQueue) incrementRetries(taskID string) {
	ptq.tasksMutex.Lock()
	defer ptq.tasksMutex.Unlock()
	
	if task, ok := ptq.tasks[taskID]; ok {
		task.Retries++
		ptq.logTask(task)
	}
}

func (ptq *PersistentTaskQueue) loadPendingTasks() error {
	file, err := os.Open(ptq.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var task Task
		if err := json.Unmarshal(scanner.Bytes(), &task); err != nil {
			continue
		}
		
		ptq.tasks[task.ID] = &task
		
		// Re-enqueue pending tasks
		if task.Status == "pending" || (task.Status == "failed" && task.Retries < 3) {
			ptq.TaskQueue.Enqueue(task.Queue, task.Data)
		}
	}
	
	return scanner.Err()
}

func (ptq *PersistentTaskQueue) GetTaskHistory(queue string, limit int) ([]*Task, error) {
	ptq.tasksMutex.RLock()
	defer ptq.tasksMutex.RUnlock()
	
	var tasks []*Task
	count := 0
	
	for _, task := range ptq.tasks {
		if queue == "" || task.Queue == queue {
			tasks = append(tasks, task)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	
	return tasks, nil
}

func (ptq *PersistentTaskQueue) GetTaskStats(queue string) map[string]int {
	ptq.tasksMutex.RLock()
	defer ptq.tasksMutex.RUnlock()
	
	stats := map[string]int{
		"pending":    0,
		"processing": 0,
		"completed":  0,
		"failed":     0,
	}
	
	for _, task := range ptq.tasks {
		if queue == "" || task.Queue == queue {
			stats[task.Status]++
		}
	}
	
	return stats
}

func (ptq *PersistentTaskQueue) Cleanup(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	
	ptq.tasksMutex.Lock()
	for id, task := range ptq.tasks {
		if task.Status == "completed" && task.Timestamp.Before(cutoff) {
			delete(ptq.tasks, id)
		}
	}
	ptq.tasksMutex.Unlock()
	
	// Rewrite log file without cleaned up tasks
	return ptq.rewriteLogFile()
}

func (ptq *PersistentTaskQueue) rewriteLogFile() error {
	tempFile := ptq.logFile + ".tmp"
	temp, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer temp.Close()
	
	ptq.tasksMutex.RLock()
	for _, task := range ptq.tasks {
		data, err := json.Marshal(task)
		if err != nil {
			continue
		}
		temp.Write(append(data, '\n'))
	}
	ptq.tasksMutex.RUnlock()
	
	temp.Close()
	
	ptq.writeMutex.Lock()
	ptq.file.Close()
	
	if err := os.Rename(tempFile, ptq.logFile); err != nil {
		ptq.writeMutex.Unlock()
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace log file: %v", err)
	}
	
	ptq.file, err = os.OpenFile(ptq.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	ptq.writeMutex.Unlock()
	
	return err
}

func (ptq *PersistentTaskQueue) Close() {
	ptq.writeMutex.Lock()
	if ptq.file != nil {
		ptq.file.Close()
	}
	ptq.writeMutex.Unlock()
	
	ptq.TaskQueue.Close()
}


func (ptq *PersistentTaskQueue) AutoCleanup(interval time.Duration, olderThan time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := ptq.Cleanup(olderThan); err != nil {
				fmt.Println("❌ Queue auto-cleanup error:", err)
			} else {
				fmt.Printf("🧹 Queue auto-cleanup done (older than %v)\n", olderThan)
			}
		}
	}()
}
