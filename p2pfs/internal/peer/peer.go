package peer

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
  "strconv"
  "context"
  "encoding/json"
  "p2pfs/internal/message"
	"path/filepath"
	"strings"
  "p2pfs/internal/utils"
  logger "p2pfs/internal/log"
  "time"
  "p2pfs/internal/fs"

)

// PeerInfo representa a un nodo en la red
type PeerInfo struct {
	ID   int
	IP   string
	Port string
}

// Peer representa al nodo local (self)
type Peer struct {
	ID      int         // ID del nodo local (0 si aún no asignado)
	IP      string      // IP local detectada
	Port    string      // Puerto en el que escucha este nodo
	Peers   []PeerInfo  // Lista de peers conocidos
	Conn    net.Conn    // Conexión TCP activa (si aplica)

	// Control de estado de descubrimiento
	LastHelloSent   time.Time // Último broadcast HELLO emitido
	LastIDAssigned  time.Time // Último momento en que recibió o asignó un ID
}

// NewPeer crea un nuevo nodo Peer
func NewPeer(id int, port string, peers []PeerInfo) *Peer {
	return &Peer{
		ID:     id,
		IP:     GetLocalIP(),
		Port:   port,
		Peers:  peers,
	}
}

func (p *Peer) AddPeer(info PeerInfo) {
	for _, existing := range p.Peers {
		if existing.IP == info.IP && existing.Port == info.Port {
			return // ya está
		}
	}
	p.Peers = append(p.Peers, info)
}


func (p *Peer) FindPeerByID(id int) *PeerInfo {
	for _, peer := range p.Peers {
		if peer.ID == id {
			return &peer
		}
	}
	return nil
}

// StartListener inicia la escucha para recibir archivos
func (p *Peer) StartListener() {
	ln, err := net.Listen("tcp", ":"+p.Port)
	if err != nil {
		fmt.Println("Error al iniciar listener:", err)
		return
	}
	defer ln.Close()

	fmt.Println("Nodo", p.ID, "escuchando en puerto", p.Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error al aceptar conexión:", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

// handleConnection recibe y guarda un archivo, verificando hash e intentando descomprimir si es ZIP
// handleConnection recibe, verifica hash y descomprime ZIPs
func (p *Peer) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Leer nombre del archivo
	filename, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error al leer nombre del archivo:", err)
		return
	}
	filename = strings.TrimSpace(filename)

	// Leer hash esperado
	expectedHash, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error al leer hash:", err)
		return
	}
	expectedHash = strings.TrimSpace(expectedHash)

	// Guardar archivo recibido
	destPath := "shared/" + filename
	file, err := os.Create(destPath)
	if err != nil {
		fmt.Println("Error al crear archivo:", err)
		return
	}
	_, err = io.Copy(file, reader)
	file.Close()
	if err != nil {
		fmt.Println("Error al guardar archivo:", err)
		return
	}

	// Registrar transferencia
	logger.AppendToLocalLog(logger.Operation{
		Type:      "TRANSFER",
		FileName:  filename,
		From:      conn.RemoteAddr().String(),
		Timestamp: time.Now().Unix(),
		Message:   "Archivo recibido",
	})

	// Verificar hash
	actualHash, err := utils.CalculateSHA256(destPath)
	if err != nil {
		fmt.Println("⚠️ Error al calcular hash:", err)
		return
	}

	if actualHash == expectedHash {
		fmt.Println("✅ Hash verificado correctamente")

		logger.AppendToLocalLog(logger.Operation{
			Type:      "HASH_OK",
			FileName:  filename,
			From:      conn.RemoteAddr().String(),
			Timestamp: time.Now().Unix(),
			Message:   "SHA256 válido",
		})
	} else {
		fmt.Println("❌ Hash inválido")
		fmt.Printf("Esperado: %s\nRecibido: %s\n", expectedHash, actualHash)

		logger.AppendToLocalLog(logger.Operation{
			Type:      "HASH_FAIL",
			FileName:  filename,
			From:      conn.RemoteAddr().String(),
			Timestamp: time.Now().Unix(),
			Message:   fmt.Sprintf("Esperado: %s, Recibido: %s", expectedHash, actualHash),
		})
		return
	}

	// Si es un ZIP, descomprimir
	if strings.HasSuffix(filename, ".zip") {
		fmt.Println("📦 ZIP detectado, descomprimiendo...")
		err := utils.UnzipFile(destPath, "shared/")
		if err != nil {
			fmt.Println("❌ Error al descomprimir:", err)

			logger.AppendToLocalLog(logger.Operation{
				Type:      "UNZIP_FAIL",
				FileName:  filename,
				From:      conn.RemoteAddr().String(),
				Timestamp: time.Now().Unix(),
				Message:   err.Error(),
			})
			return
		}
		os.Remove(destPath)
		fmt.Println("✅ Descompresión exitosa")

		logger.AppendToLocalLog(logger.Operation{
			Type:      "UNZIP",
			FileName:  filename,
			From:      conn.RemoteAddr().String(),
			Timestamp: time.Now().Unix(),
			Message:   "ZIP descomprimido correctamente",
		})
	} else {
		fmt.Println("📥 Archivo recibido como:", filename)
	}
}

// SendFile envía un archivo a un nodo destino
// SendFile comprime si es carpeta, calcula hash, y envía archivo a otro peer
// SendFile calcula hash, comprime si es carpeta y envía
func (p *Peer) SendFile(filePath, addr string) error {
	const maxRetries = 3
	const timeout = 5 * time.Second
  if p.ID == 0 {
	  return fmt.Errorf("nodo sin ID asignado, no se puede enviar archivos")
  }

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("no se pudo acceder al archivo: %v", err)
	}

	originalPath := filePath
	filename := filepath.Base(filePath)

	// Si es carpeta, crear ZIP temporal
	if info.IsDir() {
		tmpZip := filepath.Join(os.TempDir(), info.Name()+".zip")
		if err := utils.ZipFolder(filePath, tmpZip); err != nil {
			return fmt.Errorf("error al comprimir carpeta: %v", err)
		}
		filePath = tmpZip
		defer os.Remove(tmpZip)
		filename = info.Name() + ".zip"
	}

	// Calcular hash
	hash, err := utils.CalculateSHA256(filePath)
	if err != nil {
		return fmt.Errorf("error al calcular hash: %v", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("🔁 Intento %d de %d para enviar %s...\n", attempt, maxRetries, filename)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			lastErr = err
			logger.AppendToLocalLog(logger.Operation{
				Type:      "SEND_FAIL",
				FileName:  filename,
				From:      GetLocalIP() + ":" + p.Port,
				Timestamp: time.Now().Unix(),
				Message:   fmt.Sprintf("Conexión fallida (intento %d): %v", attempt, err),
			})
			time.Sleep(time.Second * time.Duration(attempt)) // Backoff
			continue
		}
		defer conn.Close()

		// Abrir el archivo para cada intento
		file, err := os.Open(filePath)
		if err != nil {
			lastErr = err
			break
		}

		// Enviar nombre y hash
		_, err = fmt.Fprintf(conn, filename+"\n"+hash+"\n")
		if err != nil {
			lastErr = err
			file.Close()
			continue
		}

		// Enviar contenido del archivo
		_, err = io.Copy(conn, file)
		file.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// Éxito
		fmt.Printf("📤 Enviado correctamente: %s → %s\n", originalPath, addr)

		logger.AppendToLocalLog(logger.Operation{
			Type:      "TRANSFER",
			FileName:  filename,
			From:      GetLocalIP() + ":" + p.Port,
			Timestamp: time.Now().Unix(),
			Message:   fmt.Sprintf("Archivo enviado exitosamente a %s en el intento %d", addr, attempt),
		})
		return nil
	}

	// Todos los intentos fallaron
	logger.AppendToLocalLog(logger.Operation{
		Type:      "SEND_FAIL",
		FileName:  filename,
		From:      GetLocalIP() + ":" + p.Port,
		Timestamp: time.Now().Unix(),
		Message:   fmt.Sprintf("Falló tras %d intentos. Último error: %v", maxRetries, lastErr),
	})

	// Agregar a la cola de reintentos
	_ = utils.AddPendingTask(utils.PendingTask{
		Type:     "TRANSFER",
		FilePath: originalPath,
		Target:   addr,
		Retries:  maxRetries,
	})

	return fmt.Errorf("falló envío tras %d intentos: %v", maxRetries, lastErr)
}

// GetLocalIP retorna la IP local de la máquina
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

func (p *Peer) RetryWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		tasks, err := utils.LoadRetryQueue()
		if err != nil {
			fmt.Println("⚠️ Error al cargar cola de reintentos:", err)
			continue
		}

		if len(tasks) == 0 {
			continue // No hay nada que hacer
		}

		fmt.Printf("🔁 Reintentando %d tarea(s) fallidas...\n", len(tasks))

		var updated []utils.PendingTask
		for _, task := range tasks {
			if task.Type != "TRANSFER" {
				updated = append(updated, task) // otros tipos, conservar
				continue
			}

			// Reintentar
			err := p.SendFile(task.FilePath, task.Target)
			if err != nil {
				// No se pudo completar, mantener en la cola
				task.Retries += 1
				updated = append(updated, task)
			}
		}

		// Reescribir la cola
		if err := utils.SaveRetryQueue(updated); err != nil {
			fmt.Println("⚠️ Error al guardar cola de reintentos:", err)
		}
	}
}

func (p *Peer) RequestFileTree(addr string) (*fs.FileNode, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	msg := message.Message{
		Type: "LIST",
		From: strconv.Itoa(p.ID),
	}
	data, _ := json.Marshal(msg)
	conn.Write(data)

	response, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	var resp message.Message
	if err := json.Unmarshal(response, &resp); err != nil {
		return nil, err
	}

	return resp.FileTree, nil
}

func (p *Peer) handleList(conn net.Conn) {
	tree, err := fs.BuildFileTree("shared")
	if err != nil {
		return
	}
	resp := message.Message{
		Type:     "LIST",
		From:     strconv.Itoa(p.ID),
		FileTree: &tree,
	}
	data, _ := json.Marshal(resp)
	conn.Write(data)
}


