package message

import "p2pfs/internal/fs"

type Message struct {
	Type      string        `json:"type"`               // TRANSFER, DELETE, LIST, etc.
	From      string        `json:"from"`               // Nodo origen
	Origin    string        `json:"origin,omitempty"`   // Nodo origen original si es relay
	FileName  string        `json:"filename,omitempty"` // Nombre del archivo
	Path      string        `json:"path,omitempty"`     // Ruta completa (sync)
	Data      []byte        `json:"data,omitempty"`     // Payload (opcional)
	FileTree  *fs.FileNode  `json:"filetree,omitempty"` // Árbol de archivos (LIST)
	Timestamp int64         `json:"timestamp"`
}

