package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var logFile = "log/oplog.json"
var mu sync.Mutex // para acceso concurrente seguro

// AppendToLocalLog agrega una operación al registro local
func AppendToLocalLog(op Operation) {
	mu.Lock()
	defer mu.Unlock()

	ops := ReadLocalLog()

	ops = append(ops, op)

	if err := saveLogToFile(ops); err != nil {
		fmt.Printf("⚠️ Error al guardar log: %v\n", err)
	}
}

// ReadLocalLog devuelve todas las operaciones registradas localmente
func ReadLocalLog() []Operation {
	mu.Lock()
	defer mu.Unlock()

	var ops []Operation

	data, err := os.ReadFile(logFile)
	if err != nil {
		// Si no existe, retornamos vacío
		return ops
	}

	if err := json.Unmarshal(data, &ops); err != nil {
		fmt.Printf("⚠️ Error al leer log: %v\n", err)
	}

	return ops
}

// saveLogToFile sobrescribe el archivo de log con el contenido dado
func saveLogToFile(ops []Operation) error {
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ops, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(logFile, data, 0644)
}
