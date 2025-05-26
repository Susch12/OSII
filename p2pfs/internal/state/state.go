package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"mod_time"`
}

type PendingTask struct {
	Type      string `json:"type"`
	FileName  string `json:"filename"`
	From      string `json:"from"`
	To        string `json:"to"`
  Retries   int    `json:"retries"`
	Timestamp int64  `json:"timestamp"`
}

type PersistentState struct {
	LastSync     map[string]int64      `json:"last_sync"`
	FileCache    map[string][]FileInfo `json:"file_cache"`
	OnlineStatus map[string]bool       `json:"online_status"`
	RetryQueue   []PendingTask         `json:"retry_queue"`
}

var (
	StateFile     = "state/state.json"
	mu            sync.Mutex
	LastSync      = make(map[string]int64)
	FileCache     = make(map[string][]FileInfo)
	OnlineStatus  = make(map[string]bool)
	RetryQueue    []PendingTask
)

// SaveState serializa el estado actual a un archivo JSON.
func SaveState() error {
	mu.Lock()
	defer mu.Unlock()

	state := PersistentState{
		LastSync:     LastSync,
		FileCache:    FileCache,
		OnlineStatus: OnlineStatus,
		RetryQueue:   RetryQueue,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	dir := filepath.Dir(StateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(StateFile, data, 0644)
}

// LoadState carga el estado desde disco si existe.
func LoadState() error {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(StateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no hay estado previo
		}
		return fmt.Errorf("read: %w", err)
	}

	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Validaciones defensivas
	if state.LastSync == nil {
		state.LastSync = make(map[string]int64)
	}
	if state.FileCache == nil {
		state.FileCache = make(map[string][]FileInfo)
	}
	if state.OnlineStatus == nil {
		state.OnlineStatus = make(map[string]bool)
	}

	LastSync = state.LastSync
	FileCache = state.FileCache
	OnlineStatus = state.OnlineStatus
	RetryQueue = state.RetryQueue
	return nil
}

// AddPendingTask agrega una tarea a la cola de reintentos.
func AddPendingTask(task PendingTask) {
	mu.Lock()
	defer mu.Unlock()
	RetryQueue = append(RetryQueue, task)
	SaveState()
}

// RemovePendingTask elimina una tarea reintentada con éxito.
func RemovePendingTask(index int) {
	mu.Lock()
	defer mu.Unlock()
	if index >= 0 && index < len(RetryQueue) {
		RetryQueue = append(RetryQueue[:index], RetryQueue[index+1:]...)
		SaveState()
	}
}

// UpdateFileCache actualiza la lista de archivos remotos para un peer.
func UpdateFileCache(peer string, files []FileInfo) {
	mu.Lock()
	defer mu.Unlock()
	FileCache[peer] = files
	LastSync[peer] = time.Now().Unix()
	SaveState()
}

// SetOnlineStatus registra el estado actual (conectado/desconectado) de un peer.
func SetOnlineStatus(peer string, online bool) {
	mu.Lock()
	defer mu.Unlock()
	OnlineStatus[peer] = online
	SaveState()
}

