package log

// Operation representa una acción sobre el sistema de archivos distribuido.
// Es usada para sincronización y registro de cambios.
type Operation struct {
	Type string // "TRANSFER" o "DELETE"
	Path string // Ruta relativa o absoluta del archivo o carpeta
	Data []byte // Contenido binario del archivo (solo para TRANSFER)
	Time int64  // Marca de tiempo Unix (para orden cronológico)
}
