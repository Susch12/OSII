package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"p2pfs/internal/log"
)

// ApplyOperation aplica una sola operación (transferencia o eliminación) al FS local.
func ApplyOperation(op log.Operation) error {
	switch op.Type {
	case "TRANSFER":
		// Crear archivo con datos
		absPath, _ := filepath.Abs(op.Path)
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creando directorio para archivo: %w", err)
		}
		err := os.WriteFile(absPath, op.Data, 0644)
		if err != nil {
			return fmt.Errorf("error al escribir archivo: %w", err)
		}
		fmt.Printf("📥 Archivo sincronizado: %s\n", absPath)

	case "DELETE":
		return DeletePath(op.Path)

	default:
		return fmt.Errorf("operación desconocida: %s", op.Type)
	}
	return nil
}

// SyncWithLogs recibe una lista de operaciones desde otros nodos
// y las aplica si son más recientes que el último timestamp local.
func SyncWithLogs(remoteLogs []log.Operation, lastSync int64) int {
	applied := 0
	for _, op := range remoteLogs {
		if op.Time > lastSync {
			if err := ApplyOperation(op); err == nil {
				log.AppendToLocalLog(op)
				applied++
			}
		}
	}
	fmt.Printf("✅ Sincronización completada. Operaciones aplicadas: %d\n", applied)
	return applied
}

// GetLastSyncTime retorna el timestamp de la última operación local.
func GetLastSyncTime() int64 {
	localLog := log.ReadLocalLog()
	var maxTime int64 = 0
	for _, op := range localLog {
		if op.Time > maxTime {
			maxTime = op.Time
		}
	}
	return maxTime
}
