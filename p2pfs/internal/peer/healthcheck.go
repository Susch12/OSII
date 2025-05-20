package peer

import (
	"fmt"
	"net"
	"time"
)

// CheckPeerAlive intenta conectarse a un peer.
// Retorna true si el nodo está activo, false si no responde.
func CheckPeerAlive(peer PeerInfo) bool {
	address := net.JoinHostPort(peer.IP, peer.Port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetLivePeers devuelve la lista de IDs de los peers que están vivos.
func GetLivePeers(peers []PeerInfo) []int {
	var alive []int
	for _, p := range peers {
		if CheckPeerAlive(p) {
			fmt.Printf("✅ Peer %d está en línea\n", p.ID)
			alive = append(alive, p.ID)
		} else {
			fmt.Printf("❌ Peer %d no responde\n", p.ID)
		}
	}
	return alive
}
