package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"p2pfs/internal/log"
)

// SaveFile guarda un archivo en el sistema de archivos local.
// Se usa cuando llega una operación TRANSFER desde otro nodo.
func SaveFile(path string, data []byte) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("no se pudo obtener path absoluto: %w", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}

	fmt.Printf("📁 Archivo guardado: %s\n", absPath)

	// Registrar operación en log
	op := log.Operation{
		Type: "TRANSFER",
		Path: absPath,
		Data: data,
		Time: time.Now().Unix(),
	}
	log.AppendToLocalLog(op)

	return nil
}
