package peer

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// PeerInfo representa a un nodo remoto en la red
type PeerInfo struct {
	ID   int
	IP   string
	Port string
}

// LocalPeer representa al nodo actual que ejecuta este código
type LocalPeer struct {
	ID    int
	Port  string
	Peers []PeerInfo
}

// NewLocalPeer crea un nuevo nodo LocalPeer con su lista de peers
func NewLocalPeer(id int, port string, peers []PeerInfo) *LocalPeer {
	return &LocalPeer{
		ID:    id,
		Port:  port,
		Peers: peers,
	}
}

// StartListener inicia la escucha para recibir archivos
func (p *LocalPeer) StartListener() {
	ln, err := net.Listen("tcp", ":"+p.Port)
	if err != nil {
		fmt.Println("❌ Error al iniciar listener:", err)
		return
	}
	defer ln.Close()

	fmt.Printf("🟢 Nodo %d escuchando en puerto %s\n", p.ID, p.Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("⚠️  Error al aceptar conexión:", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

// handleConnection recibe archivos entrantes con nombre original
func (p *LocalPeer) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Leer nombre del archivo
	filename, err := reader.ReadString('\n')
	if err == io.EOF {
		fmt.Println("⚠️ Conexión cerrada sin datos (EOF). Ignorada.")
		return
	}
	if err != nil {
		fmt.Println("⚠️ Error al leer nombre del archivo:", err)
		return
	}
	filename = strings.TrimSpace(filename)

	if filename == "" || strings.HasPrefix(filename, "recibido-") {
		fmt.Printf("⚠️ Nombre de archivo inválido o reservado: %q. Ignorado.\n", filename)
		return
	}

	// Crear archivo en carpeta shared con el nombre original
	file, err := os.Create("shared/" + filename)
	if err != nil {
		fmt.Println("⚠️  Error al crear archivo:", err)
		return
	}
	defer file.Close()

	// Leer y guardar contenido
	_, err = io.Copy(file, reader)
	if err != nil {
		fmt.Println("⚠️  Error al recibir archivo:", err)
		return
	}

	fmt.Println("📥 Archivo recibido correctamente como:", filename)
}

// SendFile envía un archivo a un nodo destino (IP:PUERTO)
func (p *LocalPeer) SendFile(filePath, addr string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo: %w", err)
	}
	defer file.Close()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("no se pudo conectar con el nodo destino: %w", err)
	}
	defer conn.Close()

	// Enviar nombre del archivo
	filename := filepath.Base(filePath)
	_, err = fmt.Fprintf(conn, filename+"\n")
	if err != nil {
		return fmt.Errorf("error al enviar nombre del archivo: %w", err)
	}

	// Enviar contenido del archivo
	_, err = io.Copy(conn, file)
	if err != nil {
		return fmt.Errorf("error al enviar el contenido del archivo: %w", err)
	}

	fmt.Println("📤 Archivo enviado correctamente:", filename)
	return nil
}
