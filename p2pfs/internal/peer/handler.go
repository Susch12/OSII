package peer

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"p2pfs/internal/fs"
	"p2pfs/internal/log"
	"p2pfs/internal/message"
	"time"
)

// StartServer inicia un servidor TCP para recibir mensajes entrantes
func StartServer(port string) {
	addr := ":" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("❌ Error al iniciar servidor: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("🛰️  Servidor escuchando en %s...\n", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("⚠️ Error al aceptar conexión: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

// handleConnection decodifica y ejecuta un mensaje entrante
func handleConnection(conn net.Conn) {
	defer conn.Close()

	data, err := io.ReadAll(conn)
	if err != nil {
		fmt.Printf("⚠️ Error al leer datos: %v\n", err)
		return
	}

	var msg message.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		fmt.Printf("⚠️ Error al parsear mensaje: %v\n", err)
		return
	}

	fmt.Printf("📩 Mensaje recibido: %s desde nodo %d\n", msg.Type, msg.Origin)

	switch msg.Type {
	case "TRANSFER":
		if err := fs.SaveFile(msg.Path, msg.Data); err != nil {
			fmt.Printf("❌ Error al guardar archivo: %v\n", err)
		}

	case "DELETE":
		if err := fs.DeletePath(msg.Path); err != nil {
			fmt.Printf("❌ Error al eliminar archivo: %v\n", err)
		} else {
			log.AppendToLocalLog(log.Operation{
				Type: "DELETE",
				Path: msg.Path,
				Time: time.Now().Unix(),
			})
		}

	case "SYNC_REQUEST":
		// Enviar nuestro log al solicitante
		ops := log.ReadLocalLog()
		payload, _ := json.Marshal(ops)
		conn.Write(payload)

	case "SYNC":
		var ops []log.Operation
		if err := json.Unmarshal(msg.Data, &ops); err != nil {
			fmt.Printf("❌ Error al parsear operaciones SYNC: %v\n", err)
			return
		}
		fs.SyncWithLogs(ops, fs.GetLastSyncTime())

	case "VIEW":
		// En versiones futuras podrías retornar vista de archivos como respuesta

	default:
		fmt.Printf("⚠️ Tipo de mensaje no soportado: %s\n", msg.Type)
	}
}
