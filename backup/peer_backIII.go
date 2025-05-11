package peer

import (
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "path/filepath"
    logfs "proyecto/internal/log" // <-- alias para evitar conflicto con log estándar
    "time"
    "encoding/json"
    "proyecto/internal/message"
)

type Peer struct {
    ID    string
    Port  string
    Peers []string
}

func NewPeer(id, port string, peers []string) *Peer {
    return &Peer{
        ID:    id,
        Port:  port,
        Peers: peers,
    }
}

func (p *Peer) StartListener() {
    ln, err := net.Listen("tcp", ":"+p.Port)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Peer %s escuchando en puerto %s", p.ID, p.Port)

    for {
        conn, err := ln.Accept()
        if err != nil {
            continue
        }
        go p.handleConnection(conn)
    }
}

func (p *Peer) handleConnection(conn net.Conn) {
    defer conn.Close()

    buf := make([]byte, 1<<20) // 1 MB máx por mensaje
    n, err := conn.Read(buf)
    if err != nil {
        log.Println("Error al leer conexión:", err)
        return
    }

    var msg message.Message
    err = json.Unmarshal(buf[:n], &msg)
    if err != nil {
        log.Println("Mensaje JSON inválido:", err)
        return
    }

    switch msg.Type {
    case "TRANSFER":
        p.handleTransfer(msg)
    case "DELETE":
        p.handleDelete(msg)
    default:
        log.Printf("Tipo de mensaje desconocido: %s", msg.Type)
    }
}

func (p *Peer) handleTransfer(msg message.Message) {
    fileName := filepath.Base(msg.FileName)
    fileName = filepath.Clean(fileName)

    outFile, err := os.Create("received_" + fileName)
    if err != nil {
        log.Println("Error al crear archivo:", err)
        return
    }
    defer outFile.Close()

    _, err = outFile.Write(msg.Payload)
    if err != nil {
        log.Println("Error al escribir archivo:", err)
        return
    }

    log.Printf("Archivo recibido: %s", fileName)

    // Registrar en log
    op := logfs.Operation{
        Type:      "TRANSFER",
        FileName:  fileName,
        From:      msg.From,
        Timestamp: msg.Timestamp,
    }
    _ = logfs.AppendOperation(op)
}



func (p *Peer) SendFile(filePath, toAddress string) error {
    conn, err := net.Dial("tcp", toAddress)
    if err != nil {
        return fmt.Errorf("Error al conectar: %v", err)
    }
    defer conn.Close()

    fileData, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("No se puede leer el archivo: %v", err)
    }

    _, fileName := filepath.Split(filePath)

    msg := message.Message{
        Type:      "TRANSFER",
        FileName:  fileName,
        Payload:   fileData,
        From:      p.ID,
        Timestamp: time.Now().Unix(),
    }

    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("Error al serializar mensaje: %v", err)
    }

    _, err = conn.Write(data)
    if err != nil {
        return fmt.Errorf("Error al enviar mensaje: %v", err)
    }

    log.Printf("Archivo %s enviado a %s", fileName, toAddress)

    // Registrar operación
    op := logfs.Operation{
        Type:      "TRANSFER",
        FileName:  fileName,
        From:      p.ID,
        Timestamp: msg.Timestamp,
    }
    _ = logfs.AppendOperation(op)

    return nil
}

func (p *Peer) BroadcastMessage(msg message.Message) {
    for _, addr := range p.Peers {
        // Evita enviar mensaje a sí mismo si la IP:puerto es el mismo
        if addr == "localhost:" + p.Port {
   	    continue
	}
        go func(to string) {
            conn, err := net.Dial("tcp", to)
            if err != nil {
                log.Printf("Error al conectar con %s: %v", to, err)
                return
            }
            defer conn.Close()

            data, err := json.Marshal(msg)
            if err != nil {
                log.Printf("Error al serializar mensaje para %s: %v", to, err)
                return
            }

            _, err = conn.Write(data)
            if err != nil {
                log.Printf("Error al enviar mensaje a %s: %v", to, err)
            } else {
                log.Printf("Mensaje %s enviado a %s", msg.Type, to)
            }
        }(addr)
    }
}


