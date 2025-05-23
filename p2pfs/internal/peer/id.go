package peer

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type NodeAnnouncement struct {
	Type string `json:"type"` // "HELLO", "ASSIGN_ID", "NEW_NODE"
	IP   string `json:"ip"`
	Port string `json:"port"`
	ID   int    `json:"id,omitempty"`
}

var (
	assignedIDs = make(map[string]int) // ip:port → ID
	nextID      = 1
	idMutex     sync.Mutex
)

// ParseAndHandleAnnouncement maneja mensajes de descubrimiento e ID
func ParseAndHandleAnnouncement(data []byte, sender *net.UDPAddr, self *Peer, getPeerList func() []PeerInfo) {
	var msg NodeAnnouncement
	if err := json.Unmarshal(data, &msg); err != nil {
		fmt.Println("⚠️ Error al parsear mensaje:", err)
		return
	}

	senderKey := net.JoinHostPort(msg.IP, msg.Port)

	switch msg.Type {

	case "HELLO":
		idMutex.Lock()
		defer idMutex.Unlock()

		if _, exists := assignedIDs[senderKey]; !exists {
			// Asignar nuevo ID al peer desconocido
			newID := getNextAvailableID()
			assignedIDs[senderKey] = newID
			fmt.Printf("🆕 Asignando ID %d a %s\n", newID, senderKey)

			// Responder directamente con ASSIGN_ID
			assignMsg := NodeAnnouncement{
				Type: "ASSIGN_ID",
				IP:   msg.IP,
				Port: msg.Port,
				ID:   newID,
			}
			sendUDPMessage(assignMsg, msg.IP)

			// Difundir NEW_NODE a todos
			BroadcastNewNode(assignMsg)
		}

	case "ASSIGN_ID":
		if self.ID == 0 && msg.IP == self.IP && msg.Port == self.Port {
			self.ID = msg.ID
			self.LastIDAssigned = time.Now()
			fmt.Printf("✅ ID %d asignado al nodo local\n", self.ID)

			// Difundir nuestra existencia
			newNode := NodeAnnouncement{
				Type: "NEW_NODE",
				IP:   self.IP,
				Port: self.Port,
				ID:   self.ID,
			}
			BroadcastNewNode(newNode)
		}

	case "NEW_NODE":
		idMutex.Lock()
		defer idMutex.Unlock()

		if _, exists := assignedIDs[senderKey]; !exists {
			assignedIDs[senderKey] = msg.ID
			fmt.Printf("📢 Nodo %s registrado con ID %d\n", senderKey, msg.ID)

			if msg.ID >= nextID {
				nextID = msg.ID + 1
			}

			// Agregar a lista de peers locales
			self.AddPeer(PeerInfo{ID: msg.ID, IP: msg.IP, Port: msg.Port})
		}
	}
}

// getNextAvailableID retorna el siguiente ID libre
func getNextAvailableID() int {
	max := 0
	for _, id := range assignedIDs {
		if id > max {
			max = id
		}
	}
	if max >= nextID {
		return max + 1
	}
	return nextID
}

// sendUDPMessage envía un mensaje UDP directo a una IP
func sendUDPMessage(msg NodeAnnouncement, ip string) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: mustParsePort(getEnvOrDefault("DISCOVERY_PORT", "48999")),
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	data, _ := json.Marshal(msg)
	conn.Write(data)
}

// BroadcastNewNode difunde un NEW_NODE por broadcast UDP
func BroadcastNewNode(msg NodeAnnouncement) {
	addr := net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: mustParsePort(getEnvOrDefault("DISCOVERY_PORT", "48999")),
	}
	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		fmt.Println("Error al emitir NEW_NODE:", err)
		return
	}
	defer conn.Close()

	data, _ := json.Marshal(msg)
	conn.Write(data)
}

// Utilidades

