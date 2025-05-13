package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "time"
    "proyecto/internal/peer"
)

func main() {
    localPort := "8001"
    knownPeers := []string{"localhost:8002"}

    localID, err := peer.LoadOrCreatePeerID()
    if err != nil {
        log.Fatalf("No se pudo cargar o crear el ID del peer: %v", err)
    }
    
    p := peer.NewPeer(localID, localPort, knownPeers)
    os.MkdirAll("shared", 0755)
    go p.StartListener()

    fmt.Println("Comandos disponibles:")
    fmt.Println(" - send <archivo> <ip:puerto>")
    fmt.Println(" - delete <archivo>")
    fmt.Println(" - sync <ip:puerto> [timestamp]")
    fmt.Println(" - list <ip:puerto>")

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "" {
            continue
        }

        args := strings.Split(input, " ")
        switch args[0] {
        case "send":
            if len(args) != 3 {
                fmt.Println("Uso: send <archivo> <ip:puerto>")
                continue
            }
            file, dest := args[1], args[2]
            if err := p.SendFile(file, dest); err != nil {
                log.Println("Error al enviar:", err)
            }

        case "delete":
            if len(args) != 2 {
                fmt.Println("Uso: delete <archivo>")
                continue
            }
            fileName := args[1]
            if err := p.DeleteFile(fileName); err != nil {
                log.Println("Error al eliminar el archivo:", err)
            }

        case "sync":
            if len(args) != 2 && len(args) != 3 {
                fmt.Println("Uso: sync <ip:puerto> [timestamp]")
                continue
            }

            dest := args[1]
            var ts int64
            var err error

            if len(args) == 3 {
                ts, err = strconv.ParseInt(args[2], 10, 64)
                if err != nil {
                    fmt.Println("Timestamp inválido. Usa un entero Unix.")
                    continue
                }
            } else {
                ts = loadLastSyncTimestamp()
                fmt.Printf("Usando timestamp anterior: %d\n", ts)
            }

            p.RequestSync(dest, ts)

            now := time.Now().Unix()
            if err := saveLastSyncTimestamp(now); err != nil {
                log.Println("No se pudo guardar last_sync.txt:", err)
            } else {
                fmt.Println("Sincronización completada. Timestamp actualizado.")
            }

        case "list":
            if len(args) != 2 {
                fmt.Println("Uso: list <ip:puerto>")
                continue
            }
            p.RequestFileTree(args[1])

        default:
            fmt.Println("Comando no reconocido.")
        }
    }
}

func saveLastSyncTimestamp(ts int64) error {
    return os.WriteFile("last_sync.txt", []byte(fmt.Sprintf("%d", ts)), 0644)
}

func loadLastSyncTimestamp() int64 {
    data, err := os.ReadFile("last_sync.txt")
    if err != nil {
        return 0 // Si no existe, asumir sincronización completa
    }

    ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
    if err != nil {
        return 0
    }

    return ts
}


