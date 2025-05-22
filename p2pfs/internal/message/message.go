package message

// Message representa un mensaje entre nodos del sistema P2P.
type Message struct {
	Type   string // "TRANSFER", "DELETE", "VIEW", "SYNC", "SYNC_REQUEST"
	Origin int    // ID del nodo que envió el mensaje
	Target int    // ID del nodo destino (0 para broadcast)
	Path   string // Ruta del archivo afectado
	Data   []byte // Contenido del archivo (para TRANSFER o SYNC)
	Time   int64  // Timestamp UNIX de la operación
}
