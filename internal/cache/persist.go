package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PersistentCache struct {
	*InMemCache
	dataFile   string
	autoSave   bool
	saveTimer  *time.Ticker
	saveMutex  sync.Mutex
}

type persistData struct {
	Store map[string]entry `json:"store"`
}

func NewPersistent(dataFile string, autoSave bool) (*PersistentCache, error) {
	inMem := NewInMem()
	
	pr := &PersistentCache{
		InMemCache: inMem,
		dataFile:   dataFile,
		autoSave:   autoSave,
	}
	
	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(dataFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Load existing data
	if err := pr.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load data: %v", err)
	}
	
	// Start auto-save if enabled
	if autoSave {
		pr.saveTimer = time.NewTicker(30 * time.Second)
		go pr.autoSaveLoop()
	}
	
	return pr, nil
}

func (pr *PersistentCache) Set(key string, value []byte, ttl time.Duration) {
	pr.InMemCache.Set(key, value, ttl)
	if pr.autoSave {
		go pr.asyncSave()
	}
}

func (pr *PersistentCache) Delete(key string) bool {
	result := pr.InMemCache.Delete(key)
	if pr.autoSave && result {
		go pr.asyncSave()
	}
	return result
}

func (pr *PersistentCache) Save() error {
	pr.saveMutex.Lock()
	defer pr.saveMutex.Unlock()
	
	pr.mu.RLock()
	data := persistData{
		Store: make(map[string]entry),
	}
	
	// Copy non-expired entries
	now := time.Now()
	for k, v := range pr.store {
		if v.ExpiresAt == nil || now.Before(*v.ExpiresAt) {
			data.Store[k] = v
		}
	}
	pr.mu.RUnlock()
	
	// Write to temp file first
	tempFile := pr.dataFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to encode data: %v", err)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, pr.dataFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename file: %v", err)
	}
	
	return nil
}

func (pr *PersistentCache) Load() error {
	file, err := os.Open(pr.dataFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	var data persistData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode data: %v", err)
	}
	
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	// Load non-expired entries
	now := time.Now()
	for k, v := range data.Store {
		if v.ExpiresAt == nil || now.Before(*v.ExpiresAt) {
			pr.store[k] = v
		}
	}
	
	return nil
}

func (pr *PersistentCache) asyncSave() {
	select {
	case <-time.After(100 * time.Millisecond):
		pr.Save()
	default:
		// Skip if already saving
	}
}

func (pr *PersistentCache) autoSaveLoop() {
	for {
		select {
		case <-pr.saveTimer.C:
			pr.Save()
		case <-pr.done:
			return
		}
	}
}

func (pr *PersistentCache) Close() {
	if pr.saveTimer != nil {
		pr.saveTimer.Stop()
	}
	
	// Final save
	pr.Save()
	
	pr.InMemCache.Close()
}
