package peer

import (
    "fmt"
    "log"
    "net"
    "os"
    "bufio"
    "path/filepath"
    "strings"
    logfs "proyecto/internal/log" // <-- alias para evitar conflicto con log estándar
    "time"
    "encoding/json"
    "proyecto/internal/message"
)
const SharedDir = "shared"
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
    case "SYNC":
	    p.handleSync(msg, conn)
    case "LIST":
      p.handleList(msg, conn)
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

func (p *Peer) handleSync(msg message.Message, conn net.Conn) {
    ops, err := logfs.ReplayLog(msg.Timestamp)
    if err != nil {
        log.Println("Error al leer log para sincronización:", err)
        return
    }

    for _, op := range ops {
        syncMsg := message.Message{
            Type:      op.Type,
            FileName:  op.FileName,
            From:      op.From,
            Timestamp: op.Timestamp,
        }

        data, err := json.Marshal(syncMsg)
        if err != nil {
            log.Println("Error al serializar syncMsg:", err)
            continue
        }

        _, err = conn.Write(append(data, '\n'))
        if err != nil {
            log.Println("Error al enviar operación de sincronización:", err)
            return
        }
    }

    log.Printf("Enviadas %d operaciones de sincronización a %s", len(ops), msg.From)
}

func (p *Peer) RequestSync(fromAddress string, since int64) {
    conn, err := net.Dial("tcp", fromAddress)
    if err != nil {
        log.Printf("No se pudo conectar a %s para sincronizar: %v", fromAddress, err)
        return
    }
    defer conn.Close()

    msg := message.Message{
        Type:      "SYNC",
        From:      p.ID,
        Timestamp: since,
    }

    data, err := json.Marshal(msg)
    if err != nil {
        log.Println("Error al serializar SYNC:", err)
        return
    }

    _, err = conn.Write(data)
    if err != nil {
        log.Println("Error al enviar mensaje SYNC:", err)
        return
    }

    // Leer múltiples mensajes (uno por operación)
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        var op message.Message
        if err := json.Unmarshal(scanner.Bytes(), &op); err != nil {
            log.Println("Operación de sincronización inválida:", err)
            continue
        }

        switch op.Type {
        case "TRANSFER":
            p.handleTransfer(op)
        case "DELETE":
            p.handleDelete(op)
        }
    }

    if err := scanner.Err(); err != nil {
        log.Println("Error al leer respuesta de sincronización:", err)
    }
}

func (p *Peer) DeleteFile(fileName string) error {
    cleanName := filepath.Base(fileName)
    cleanName = filepath.Clean(cleanName)

    err := os.Remove("received_" + cleanName)
    if err != nil {
        return fmt.Errorf("no se pudo eliminar el archivo: %v", err)
    }

    log.Printf("Archivo eliminado localmente: %s", cleanName)

    msg := message.Message{
        Type:      "DELETE",
        FileName:  cleanName,
        From:      p.ID,
        Timestamp: time.Now().Unix(),
    }

    op := logfs.Operation{
        Type:      "DELETE",
        FileName:  cleanName,
        From:      p.ID,
        Timestamp: msg.Timestamp,
    }

    _ = logfs.AppendOperation(op)
    p.BroadcastMessage(msg)

    return nil
}

func (p *Peer) handleDelete(msg message.Message) {
    cleanName := filepath.Base(msg.FileName)
    cleanName = filepath.Clean(cleanName)

    err := os.Remove("received_" + cleanName)
    if err != nil {
        log.Printf("Error al eliminar archivo %s: %v", cleanName, err)
        // No return: aún así se registra en el log
    } else {
        log.Printf("Archivo eliminado remotamente: %s", cleanName)
    }

    op := logfs.Operation{
        Type:      "DELETE",
        FileName:  cleanName,
        From:      msg.From,
        Timestamp: msg.Timestamp,
    }

    _ = logfs.AppendOperation(op)
}

func buildFileTree(path string) (*message.FileNode, error) {
    fi, err := os.Stat(path)
    if err != nil {
        return nil, err
    }

    node := &message.FileNode{
        Name:  fi.Name(),
        IsDir: fi.IsDir(),
    }

    if fi.IsDir() {
        entries, err := os.ReadDir(path)
        if err != nil {
            return nil, err
        }

        for _, entry := range entries {
            childPath := filepath.Join(path, entry.Name())
            childNode, err := buildFileTree(childPath)
            if err != nil {
                continue
            }
            node.Children = append(node.Children, *childNode)
        }
    }

    return node, nil
}

func (p *Peer) handleList(msg message.Message, conn net.Conn) {
    tree, err := buildFileTree(SharedDir)
    if err != nil {
        log.Println("Error al construir árbol:", err)
        return
    }

    response := message.Message{
        Type:      "LIST",
        From:      p.ID,
        Tree:      tree,
        Timestamp: time.Now().Unix(),
    }

    data, _ := json.Marshal(response)
    conn.Write(data)
}


func (p *Peer) RequestFileTree(addr string) {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        log.Println("No se pudo conectar:", err)
        return
    }
    defer conn.Close()

    msg := message.Message{
        Type:      "LIST",
        From:      p.ID,
        Timestamp: time.Now().Unix(),
    }

    data, _ := json.Marshal(msg)
    conn.Write(data)

    // Esperar respuesta (estructura del árbol)
    buf := make([]byte, 1<<20)
    n, _ := conn.Read(buf)

    var resp message.Message
    if err := json.Unmarshal(buf[:n], &resp); err != nil {
        log.Println("Respuesta inválida:", err)
        return
    }

    if resp.Tree != nil {
        printFileTree(*resp.Tree, 0)
    }
}

func printFileTree(node message.FileNode, depth int) {
    prefix := strings.Repeat("  ", depth)
    if node.IsDir {
        fmt.Printf("%s📁 %s\n", prefix, node.Name)
        for _, child := range node.Children {
            printFileTree(child, depth+1)
        }
    } else {
        fmt.Printf("%s📄 %s\n", prefix, node.Name)
    }
}
