package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PeerInfo struct {
	ID   int    `json:"id"`
	IP   string `json:"ip"`
	Port string `json:"port"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run id.go <puerto>")
		return
	}
	port := os.Args[1]
	RegisterSelf(port)
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func RegisterSelf(port string) {
	idPath := "peer.id"
	jsonPath := filepath.Join("config", "peers.json")

	data, err := os.ReadFile(idPath)
	if err != nil {
		fmt.Println("❌ No se pudo leer peer.id:", err)
		return
	}
	id, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Println("❌ ID inválido en peer.id:", err)
		return
	}

	ip := GetLocalIP()

	// Leer peers.json
	var peers []PeerInfo
	if content, err := os.ReadFile(jsonPath); err == nil {
		json.Unmarshal(content, &peers)
	}

	updated := false
	for i := range peers {
		if peers[i].ID == id {
			peers[i].IP = ip
			peers[i].Port = port
			updated = true
			break
		}
	}
	if !updated {
		peers = append(peers, PeerInfo{ID: id, IP: ip, Port: port})
	}

	out, err := json.MarshalIndent(peers, "", "  ")
	if err != nil {
		fmt.Println("❌ Error al generar JSON:", err)
		return
	}

	if err := os.WriteFile(jsonPath, out, 0644); err != nil {
		fmt.Println("❌ No se pudo escribir config/peers.json:", err)
	} else {
		fmt.Printf("✅ Nodo ID %d registrado como %s:%s en %s\n", id, ip, port, jsonPath)
	}
}
