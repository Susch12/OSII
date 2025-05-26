// main.go - Lanzador principal con autoinclusión del nodo local
package main

import (
	"fmt"
	"p2pfs/internal/gui"
	"p2pfs/internal/peer"
	"time"
)

func main() {
	// 🛠 Configuración inicial
	port := "8001"
	localIP := peer.GetLocalIP()
	fmt.Println("Esta máquina tiene IP:", localIP)

	// Crear nodo sin ID asignado aún
	self := &peer.Peer{
		ID:    0,
		IP:    localIP,
		Port:  port,
		Peers: []peer.PeerInfo{},
	}

	// 🔊 Listeners y tareas de red
	go peer.ListenForBroadcasts(self, func() []peer.PeerInfo {
		return self.Peers
	})
	go peer.BroadcastHello(self)
	go self.StartListener()
	go self.RetryWorker(10 * time.Second)
	go self.MonitorPeersAndSync(5 * time.Second)

	// ⏱️ Esperar ID o asignarlo
	time.Sleep(5 * time.Second)
	if self.ID == 0 {
		fmt.Println("⚠️  No se recibió ASSIGN_ID. Asignando ID=1 como nodo inicial.")
		self.ID = 1
		self.LastIDAssigned = time.Now()

		// 👤 Agregar el propio nodo como peer visible
		self.Peers = append(self.Peers, peer.PeerInfo{
			ID:   self.ID,
			IP:   self.IP,
			Port: self.Port,
		})

		// 🔈 Anunciar el nodo
		newNode := peer.NodeAnnouncement{
			Type: "NEW_NODE",
			IP:   self.IP,
			Port: self.Port,
			ID:   self.ID,
		}
		peer.BroadcastNewNode(newNode)
	}

	// ✅ Asegurar inclusión propia si ya fue asignado por otro
	alreadyPresent := false
	for _, p := range self.Peers {
		if p.IP == self.IP && p.Port == self.Port {
			alreadyPresent = true
			break
		}
	}
	if !alreadyPresent {
		self.Peers = append(self.Peers, peer.PeerInfo{
			ID:   self.ID,
			IP:   self.IP,
			Port: self.Port,
		})
	}

	// 🖼️ Lanzar GUI con información válida
	fmt.Println("🟢 Lanzando GUI...")
	gui.StartGUI(self.ID, func() []peer.PeerInfo {
		return self.Peers
	}, self)
}

