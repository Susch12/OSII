package message

// Tipos de mensaje válidos para la red P2P
const (
	TypeTransfer = "TRANSFER"
	TypeDelete   = "DELETE"
	TypeSync     = "SYNC"
	TypeList     = "LIST"
)

// FileNode representa un nodo (archivo o directorio) en la estructura del sistema de archivos
type FileNode struct {
	Name     string     `json:"name"`                // Nombre del archivo o directorio
	IsDir    bool       `json:"is_dir"`              // True si es un directorio
	Children []FileNode `json:"children,omitempty"`  // Subnodos (sólo si es directorio)
}

// Message representa un mensaje intercambiado entre peers en la red
type Message struct {
	Type      string     `json:"type"`                // Tipo de mensaje: TRANSFER, DELETE, SYNC, LIST
	FileName  string     `json:"filename,omitempty"`  // Nombre del archivo (si aplica)
	Payload   []byte     `json:"payload,omitempty"`   // Contenido del archivo (TRANSFER)
	From      string     `json:"from"`                // ID del peer que envía el mensaje
	Timestamp int64      `json:"timestamp"`           // Marca de tiempo en formato Unix
	Tree      *FileNode  `json:"tree,omitempty"`      // Árbol de archivos (respuesta a LIST)
	Error     string     `json:"error,omitempty"`     // Mensaje de error opcional (para respuestas con fallos)
}

