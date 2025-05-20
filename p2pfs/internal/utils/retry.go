package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var retryFile = "log/retry_queue.json"
var mu sync.Mutex

// PendingTask representa una tarea que no se pudo completar (ej. TRANSFER, DELETE)
type PendingTask struct {
	Type     string `json:"type"`     // Ej. "TRANSFER", "DELETE"
	FilePath string `json:"filepath"` // Ruta local del archivo
	Target   string `json:"target"`   // IP:puerto destino
	Retries  int    `json:"retries"`  // Número de intentos fallidos previos
}

// AddPendingTask agrega una nueva tarea a retry_queue.json
func AddPendingTask(task PendingTask) error {
	mu.Lock()
	defer mu.Unlock()

	tasks, _ := LoadRetryQueue()
	tasks = append(tasks, task)

	if err := saveRetryQueue(tasks); err != nil {
		return fmt.Errorf("error al guardar retry_queue: %v", err)
	}
	return nil
}

// LoadRetryQueue lee todas las tareas pendientes
func LoadRetryQueue() ([]PendingTask, error) {
	var tasks []PendingTask

	data, err := os.ReadFile(retryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return tasks, nil // no hay archivo, no hay tareas
		}
		return nil, fmt.Errorf("error al leer retry_queue: %v", err)
	}

	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("formato inválido en retry_queue: %v", err)
	}

	return tasks, nil
}

// saveRetryQueue sobrescribe el archivo con la nueva lista de tareas
func saveRetryQueue(tasks []PendingTask) error {
	dir := filepath.Dir(retryFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(retryFile, data, 0644)
}

// SaveRetryQueue guarda la lista actualizada de tareas
func SaveRetryQueue(tasks []PendingTask) error {
	mu.Lock()
	defer mu.Unlock()
	return saveRetryQueue(tasks)
}
