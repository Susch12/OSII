package main

import (
	"fmt"
	"p2pfs/internal/gui"
	"p2pfs/internal/peer"
)

func main() {
	// Cambia estos valores por nodo
	selfID := 1
	port := "8001"

	localIP := peer.GetLocalIP()
	fmt.Println("Esta máquina tiene IP:", localIP)



	// Lista de todos los nodos (IPs REALES en la red)
	peers := []peer.PeerInfo{
		{ID: 1, IP: "192.168.0.6", Port: "8001"},
		{ID: 2, IP: "192.168.1.11", Port: "8002"},
		{ID: 3, IP: "192.168.1.12", Port: "8003"},
		{ID: 4, IP: "192.168.1.13", Port: "8004"},
	}

	self := peer.NewPeer(selfID, port, peers)

	go self.StartListener()
  // ✅ Lanzar el RetryWorker cada 10 segundos
	go self.RetryWorker(10 * time.Second)

	gui.StartGUI(selfID, peers, self)
}

