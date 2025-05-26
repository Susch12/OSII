package fs

import (
	"os"
	"path/filepath"
	"time"
)

// FileInfo representa la información de un archivo o carpeta.
type FileInfo struct {
	Name     string    `json:"name"`
	FullPath string    `json:"full_path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	IsDir    bool      `json:"is_dir"`
}

// ListFiles recorre el sistema de archivos local (desde una ruta base) y
// retorna la lista de archivos y carpetas con su información relevante.
func ListFiles(baseDir string) ([]FileInfo, error) {
	var result []FileInfo

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// No se puede acceder al archivo, continuar
			return nil
		}

		// Saltar archivos ocultos o temporales si se desea
		// if strings.HasPrefix(info.Name(), ".") {
		// 	return nil
		// }

		entry := FileInfo{
			Name:     info.Name(),
			FullPath: path,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			IsDir:    info.IsDir(),
		}
		result = append(result, entry)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
