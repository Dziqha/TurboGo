package pubsub

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Message struct {
	Topic     string    `json:"topic"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
}

type PersistentEventBus struct {
	*EventBus
	logFile     string
	file        *os.File
	writeMutex  sync.Mutex
	replayMode  bool
	messageID   uint64
	idMutex     sync.Mutex
}

func NewPersistent(logFile string) (*PersistentEventBus, error) {
	inMem := NewInMem()
	
	peb := &PersistentEventBus{
		EventBus:  inMem,
		logFile:   logFile,
		messageID: 1,
	}

	peb.AutoCompact(10 * time.Minute)
	
	return peb, nil
}

func (p *PersistentEventBus) ensureFile() error {
	if p.file != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p.logFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	file, err := os.OpenFile(p.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	p.file = file
	return nil
}


func (peb *PersistentEventBus) Publish(topic string, msg []byte) error {
	if err := peb.ensureFile(); err != nil {
		return err
	}
	peb.idMutex.Lock()
	id := fmt.Sprintf("%d", peb.messageID)
	peb.messageID++
	peb.idMutex.Unlock()
	
	message := Message{
		Topic:     topic,
		Data:      msg,
		Timestamp: time.Now(),
		ID:        id,
	}
	
	if err := peb.logMessage(message); err != nil {
		return fmt.Errorf("failed to log message: %v", err)
	}
	
	if !peb.replayMode {
		return peb.EventBus.Publish(topic, msg)
	}
	
	return nil
}

func (peb *PersistentEventBus) logMessage(msg Message) error {
	peb.writeMutex.Lock()
	defer peb.writeMutex.Unlock()
	
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	_, err = peb.file.Write(append(data, '\n'))
	if err != nil {
		return err
	}
	
	return peb.file.Sync()
}

func (peb *PersistentEventBus) Replay(fromTime *time.Time, toTime *time.Time) error {
	peb.replayMode = true
	defer func() { peb.replayMode = false }()
	
	file, err := os.Open(peb.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil 
		}
		return fmt.Errorf("failed to open log file for replay: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue 
		}
		
		if fromTime != nil && msg.Timestamp.Before(*fromTime) {
			continue
		}
		if toTime != nil && msg.Timestamp.After(*toTime) {
			continue
		}
		
		peb.EventBus.Publish(msg.Topic, msg.Data)
	}
	
	return scanner.Err()
}

func (peb *PersistentEventBus) GetMessageHistory(topic string, limit int) ([]Message, error) {
	file, err := os.Open(peb.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()
	
	var messages []Message
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		
		if topic == "" || msg.Topic == topic {
			messages = append(messages, msg)
			if limit > 0 && len(messages) >= limit {
				break
			}
		}
	}
	
	return messages, scanner.Err()
}

func (peb *PersistentEventBus) Compact() error {
	tempFile := peb.logFile + ".tmp"
	temp, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer temp.Close()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	messages, err := peb.GetMessageHistory("", 0)
	if err != nil {
		os.Remove(tempFile)
		return err
	}
	
	for _, msg := range messages {
		if msg.Timestamp.After(cutoff) {
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			temp.Write(append(data, '\n'))
		}
	}
	
	temp.Close()
	
	peb.writeMutex.Lock()
	peb.file.Close()
	
	if err := os.Rename(tempFile, peb.logFile); err != nil {
		peb.writeMutex.Unlock()
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace log file: %v", err)
	}
	
	peb.file, err = os.OpenFile(peb.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	peb.writeMutex.Unlock()

	fmt.Printf("🧹 Pubsub compact done: %d messages kept\n", len(messages))
	
	return err
}


func (peb *PersistentEventBus) AutoCompact(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := peb.Compact(); err != nil {
				fmt.Println("❌ Pubsub compact error:", err)
			}
		}
	}()
}


func (peb *PersistentEventBus) Close() {
	peb.writeMutex.Lock()
	if peb.file != nil {
		peb.file.Close()
	}
	peb.writeMutex.Unlock()
	
	peb.EventBus.Close()
}