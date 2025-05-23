package fs

import (
	"os"
	"path/filepath"
	"time"
)

// FileNode representa un nodo en el árbol de archivos.
type FileNode struct {
	Name     string     `json:"name"`               // Nombre del archivo o carpeta
	IsDir    bool       `json:"is_dir"`             // Si es directorio
	ModTime  time.Time  `json:"mod_time"`           // Última modificación
	Children []FileNode `json:"children,omitempty"` // Hijos (si es directorio)
}

// BuildFileTree construye recursivamente un árbol desde un directorio base.
func BuildFileTree(root string) (FileNode, error) {
	info, err := os.Stat(root)
	if err != nil {
		return FileNode{}, err
	}

	node := FileNode{
		Name:    info.Name(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
	}

	if !info.IsDir() {
		return node, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return node, err
	}

	for _, entry := range entries {
		childPath := filepath.Join(root, entry.Name())
		childNode, err := BuildFileTree(childPath)
		if err == nil {
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

