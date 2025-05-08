package message

type Message struct {
    Type      string `json:"type"`                // TRANSFER, DELETE, SYNC
    FileName  string `json:"filename,omitempty"`  // archivo a transferir/eliminar
    Payload   []byte `json:"payload,omitempty"`   // contenido del archivo (solo para TRANSFER)
    From      string `json:"from"`                // ID del peer que envió el mensaje
    Timestamp int64  `json:"timestamp"`           // Unix timestamp
}


