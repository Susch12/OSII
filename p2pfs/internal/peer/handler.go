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
        // Aquí iría el manejo de transferencia si se implementa
        fmt.Println("🔄 TRANSFER recibido (aún no implementado)")

    case "DELETE":
        // Aquí iría el manejo de eliminación si se implementa
        fmt.Println("🗑️ DELETE recibido (aún no implementado)")

    case "LIST":
        fmt.Printf("📦 Respondiendo a solicitud LIST desde nodo %d\n", msg.Origin)

        // Generar el árbol de archivos
        tree := fs.GenerateFileTree("shared")

        // Preparar respuesta
        response := msg
        response.Tree = tree

        // Codificar y enviar
        encoder := json.NewEncoder(conn)
        if err := encoder.Encode(response); err != nil {
            fmt.Printf("❌ Error al enviar lista: %v\n", err)
        }

    default:
        fmt.Printf("❓ Tipo de mensaje no reconocido: %s\n", msg.Type)
    }
}
