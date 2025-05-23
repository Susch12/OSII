package fs

import (
	"os"
)

// DeleteFile elimina un archivo específico
func DeleteFile(path string) error {
	return os.Remove(path)
}

// DeletePath elimina un archivo o carpeta (de forma recursiva si es directorio)
func DeletePath(path string) error {
	return os.RemoveAll(path)
}
