package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strings"

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
    go p.StartListener()

    fmt.Println("Comandos disponibles:")
    fmt.Println(" - send <archivo> <ip:puerto>")
    fmt.Println(" - (más comandos próximamente)")

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
        default:
            fmt.Println("Comando no reconocido.")
        }
    }
}

