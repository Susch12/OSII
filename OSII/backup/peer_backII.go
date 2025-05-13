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

    name := make([]byte, 256)
    n, err := conn.Read(name)
    if err != nil {
        log.Println("Error al leer nombre:", err)
        return
    }

    fileName := string(name[:n])
    fileName = filepath.Base(fileName)
    fileName = filepath.Clean(fileName)

    log.Printf("Recibiendo archivo: %s", fileName)

    outFile, err := os.Create("received_" + fileName)
    if err != nil {
        log.Println("Error al crear archivo:", err)
        return
    }
    defer outFile.Close()

    bytesWritten, err := io.Copy(outFile, conn)
    if err != nil {
        log.Println("Error al recibir archivo:", err)
        return
    }

    log.Printf("Archivo recibido %s (%d bytes)", fileName, bytesWritten)
}

func (p *Peer) SendFile(filePath, toAddress string) error {
    conn, err := net.Dial("tcp", toAddress)
    if err != nil {
        return fmt.Errorf("Error al conectar: %v", err)
    }
    defer conn.Close()

    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("No se puede abrir el archivo: %v", err)
    }
    defer file.Close()

    _, fileName := filepath.Split(filePath)
    _, err = conn.Write([]byte(fileName))
    if err != nil {
        return fmt.Errorf("Error al enviar nombre: %v", err)
    }

    _, err = io.Copy(conn, file)
    if err != nil {
        return fmt.Errorf("Error al enviar archivo: %v", err)
    }

    log.Printf("Archivo %s enviado a %s", fileName, toAddress)

    // ✅ Registrar la operación en el log
    op := logfs.Operation{
        Type:      "TRANSFER",
        FileName:  fileName,
        From:      p.ID,
        Timestamp: time.Now().Unix(),
    }
    _ = logfs.AppendOperation(op)

    return nil
}

