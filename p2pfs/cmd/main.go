package main

import (
	"fmt"
	"p2pfs/internal/gui"
	"p2pfs/internal/peer"
	"time"
)

func main() {
	// Configuración inicial sin ID (se asignará dinámicamente)
	port := "8001"
	localIP := peer.GetLocalIP()
	fmt.Println("Esta máquina tiene IP:", localIP)

	// Crear nodo sin ID (será asignado luego)
	self := &peer.Peer{
		ID:    0, // ID aún no asignado
		IP:    localIP,
		Port:  port,
		Peers: []peer.PeerInfo{},
	}

	// 🔊 Listener para handshakes y mensajes UDP
	go peer.ListenForBroadcasts(self, func() []peer.PeerInfo {
		return self.Peers
	})

	// 📣 Broadcast activo mientras no tenga ID
	go peer.BroadcastHello(self)

	// 🧠 Iniciar listener TCP de archivos, SYNC, etc.
	go self.StartListener()

	// ♻️ Reintentos de envío de archivos fallidos
	go self.RetryWorker(10 * time.Second)
  // Si después de 5 segundos no se ha recibido ID, autoasignar
  go func() {

	  time.Sleep(5 * time.Second)
	  if self.ID == 0 {
		  fmt.Println("⚠️  No se recibió ASSIGN_ID. Asignando ID=1 como nodo inicial.")
		  self.ID = 1
		  self.LastIDAssigned = time.Now()

		  newNode := peer.NodeAnnouncement{
			  Type: "NEW_NODE",
			  IP:   self.IP,
			  Port: self.Port,
			  ID:   self.ID,
		  }
		  peer.BroadcastNewNode(newNode)
	  }
  }()

	// 🖼️ Interfaz gráfica
	gui.StartGUI(self.ID, self.Peers, self)
}

