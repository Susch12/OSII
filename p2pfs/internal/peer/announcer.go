package peer

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

var BroadcastPort = getEnvOrDefault("DISCOVERY_PORT", "48999")
const BroadcastInterval = 5 * time.Second

// BroadcastHello emite periódicamente un mensaje HELLO por UDP broadcast
func BroadcastHello(self *Peer) {
	addr := net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: mustParsePort(BroadcastPort),
	}
	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		fmt.Println("Error al emitir broadcast HELLO:", err)
		return
	}
	defer conn.Close()

	for {
		if self.ID == 0 {
			msg := NodeAnnouncement{
				Type: "HELLO",
				IP:   self.IP,
				Port: self.Port,
			}
			data, _ := json.Marshal(msg)

			_, err := conn.Write(data)
			if err == nil {
				self.LastHelloSent = time.Now()
				fmt.Println("📣 Enviado HELLO desde", self.IP+":"+self.Port)
			}
		}
		time.Sleep(BroadcastInterval)
	}
}

// ListenForBroadcasts escucha mensajes por UDP (HELLO, ASSIGN_ID, NEW_NODE)
func ListenForBroadcasts(self *Peer, getPeerList func() []PeerInfo) {
	addr := net.UDPAddr{
		IP:   net.IPv4zero,
		Port: mustParsePort(BroadcastPort),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error al escuchar broadcast:", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, sender, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		go handleBroadcastMessage(buf[:n], sender, self, getPeerList)
	}
}

func handleBroadcastMessage(data []byte, sender *net.UDPAddr, self *Peer, getPeerList func() []PeerInfo) {
	ParseAndHandleAnnouncement(data, sender, self, getPeerList)
}

// Utilidades

func mustParsePort(p string) int {
	port, err := strconv.Atoi(p)
	if err != nil {
		fmt.Printf("Error al convertir el puerto: %s\n", p)
		return 48999
	}
	return port
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
