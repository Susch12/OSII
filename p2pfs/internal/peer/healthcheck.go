package peer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// Última vez que un peer respondió
var peerLastSeen = make(map[string]time.Time)

// Estado anterior de cada peer (activo/inactivo)
var peerWasDown = make(map[string]bool)


type HandshakeMessage struct {
	Type       string   `json:"type"`
	From       string   `json:"from"`
	KnownPeers []string `json:"known_peers,omitempty"`
}

// Mapa de estado de peers vivos
var peerStatuses = make(map[string]bool)

// CheckPeerAlive intenta conectarse a un peer
func CheckPeerAlive(peer PeerInfo) bool {
	address := net.JoinHostPort(peer.IP, peer.Port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		peerStatuses[address] = false
		return false
	}
	conn.Close()
	peerStatuses[address] = true
	return true
}

// GetLivePeers retorna los IDs de peers activos
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

// StartHandshakeListener inicia el servidor de descubrimiento
func StartHandshakeListener(self PeerInfo, getPeerList func() []PeerInfo) {
	ln, err := net.Listen("tcp", net.JoinHostPort(self.IP, self.Port))
	if err != nil {
		fmt.Println("Error al iniciar listener:", err)
		return
	}
	fmt.Println("🔊 Escuchando handshakes en", self.IP+":"+self.Port)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go handleHandshake(conn, self, getPeerList)
		}
	}()
}

func handleHandshake(conn net.Conn, self PeerInfo, getPeerList func() []PeerInfo) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	var msg HandshakeMessage
	json.Unmarshal([]byte(line), &msg)

	if msg.Type == "HELLO" {
		known := getPeerList()
		var knownStrs []string
		for _, p := range known {
			knownStrs = append(knownStrs, net.JoinHostPort(p.IP, p.Port))
		}
		response := HandshakeMessage{
			Type:       "WELCOME",
			From:       net.JoinHostPort(self.IP, self.Port),
			KnownPeers: knownStrs,
		}
		resBytes, _ := json.Marshal(response)
		conn.Write(append(resBytes, '\n'))
	}
}

// SendHelloAndReceivePeers envía HELLO y recibe WELCOME
func SendHelloAndReceivePeers(addr string) ([]string, error) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	msg := HandshakeMessage{
		Type: "HELLO",
		From: "?", // opcional
	}

data, _ := json.Marshal(msg)
	conn.Write(append(data, '\n'))

	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')

	var res HandshakeMessage
	json.Unmarshal([]byte(line), &res)

	if res.Type == "WELCOME" {
		return res.KnownPeers, nil
	}
	return nil, fmt.Errorf("respuesta inválida del handshake")
}

// MergePeerListsFromStrings añade nuevos peers evitando duplicados
func MergePeerListsFromStrings(known []string, current *[]PeerInfo) {
	existing := make(map[string]bool)
	for _, p := range *current {
		existing[net.JoinHostPort(p.IP, p.Port)] = true
	}
	for _, addr := range known {
		if !existing[addr] {
			ipPort := strings.Split(addr, ":")
			if len(ipPort) == 2 {
				*current = append(*current, PeerInfo{IP: ipPort[0], Port: ipPort[1]})
			}
		}
	}
}

// UpdatePeerStatus verifica si el peer ha reconectado recientemente.
// Si es así, retorna true (debe sincronizarse).
func UpdatePeerStatus(peer PeerInfo) bool {
	address := net.JoinHostPort(peer.IP, peer.Port)
	alive := CheckPeerAlive(peer)

	if alive {
		now := time.Now()
		_, seen := peerLastSeen[address]

		// Primera vez o antes estaba caído
		if !seen || peerWasDown[address] {
			peerLastSeen[address] = now
			peerWasDown[address] = false
			fmt.Printf("🔄 Reconexión detectada: %s\n", address)
			return true // ← señal para iniciar sincronización
		}

		// Actualización normal
		peerLastSeen[address] = now
		peerWasDown[address] = false
		return false
	} else {
		peerWasDown[address] = true
		return false
	}
}

// MonitorPeersAndSync revisa periódicamente el estado de los peers
// y sincroniza automáticamente si detecta reconexión.
func (p *Peer) MonitorPeersAndSync(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, peerInfo := range p.Peers {
			// No sincronizamos con nosotros mismos
			if peerInfo.Port == p.Port && peerInfo.IP == p.IP {
				continue
			}

			if UpdatePeerStatus(peerInfo) {
				fmt.Printf("📡 Iniciando sincronización con %s:%s...\n", peerInfo.IP, peerInfo.Port)
				go p.SyncWithPeer(peerInfo)
			}
		}
	}
}

