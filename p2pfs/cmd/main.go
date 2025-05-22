package main

import (
	"fmt"
	"log"

	"p2pfs/internal/gui"
	"p2pfs/internal/peer"
)

func main() {
	// 1. Cargar lista de peers desde el archivo JSON
	peers, err := peer.LoadPeers("config/peers.json")
	if err != nil {
		log.Fatalf("❌ Error al cargar peers.json: %v", err)
	}

	// 2. Detectar IP local y obtener ID
	localID, err := peer.GetLocalID(peers)
	if err != nil {
		log.Fatalf("❌ Error al detectar ID local: %v", err)
	}

	// 3. Convertir []peer.Peer a []peer.PeerInfo
	var peerInfoList []peer.PeerInfo
	for _, p := range peers {
		peerInfoList = append(peerInfoList, peer.PeerInfo{
			ID:   p.ID,
			IP:   p.IP,
			Port: p.Port,
		})
	}

	// 4. Obtener datos del nodo actual (su IP y puerto)
	var self peer.Peer
	for _, p := range peers {
		if p.ID == localID {
			self = p
			break
		}
	}

	// 5. Mostrar información del nodo
	fmt.Printf("✅ Nodo activo:\n")
	fmt.Printf("   ID   : %d\n", self.ID)
	fmt.Printf("   IP   : %s\n", self.IP)
	fmt.Printf("   Puerto: %s\n", self.Port)

	// 6. Crear el nodo LocalPeer
	localPeer := peer.NewLocalPeer(self.ID, self.Port, peerInfoList)

	// 7. Iniciar el listener del nodo (en segundo plano)
	go localPeer.StartListener()

	// 8. Abrir GUI
	gui.StartGUI(self.ID, peerInfoList, localPeer)
}
