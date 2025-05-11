package main

import (
    "log"
    "bufio"
    "fmt"
    "os"
    "proyecto/internal/peer"
)


func main() {
    localPort := "8001"
    peers := []string{"localhost:8002"}
    p := peer.NewPeer("peer1", localPort, peers)

    go p.StartListener()

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("Comando (send <archivo> <ip:puerto>): ")
        scanner.Scan()
        input := scanner.Text()

        var file, dest string
        n, _ := fmt.Sscanf(input, "send %s %s", &file, &dest)
        if n == 2 {
            err := p.SendFile(file, dest)
            if err != nil {
                log.Println("Error:", err)
            }
        }
    }
}

