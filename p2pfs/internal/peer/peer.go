package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"p2pfs/internal/fs"
	logger "p2pfs/internal/log"
	"p2pfs/internal/message"
	"p2pfs/internal/state"
	"p2pfs/internal/utils"
	"strconv"
	"strings"
	"time"
)

type PeerInfo struct {
	ID   int
	IP   string
	Port string
}

type Peer struct {
	ID             int
	IP             string
	Port           string
	Peers          []PeerInfo
	Conn           net.Conn
	LastHelloSent  time.Time
	LastIDAssigned time.Time
}

func NewPeer(id int, port string, peers []PeerInfo) *Peer {
	return &Peer{
		ID:    id,
		IP:    GetLocalIP(),
		Port:  port,
		Peers: peers,
	}
}

func (p *Peer) AddPeer(info PeerInfo) {
	for _, existing := range p.Peers {
		if existing.IP == info.IP && existing.Port == info.Port {
			return
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


func (p *Peer) handleConnection(conn net.Conn) {
	defer conn.Close()

	data, err := io.ReadAll(conn)
	if err != nil {
		fmt.Println("❌ Error al leer conexión:", err)
		return
	}

	if len(data) == 0 {
		fmt.Println("⚠️ Conexión recibida sin datos. Ignorando.")
		return
	}

	var msg message.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		fmt.Println("⚠️ Entrada no válida como JSON. Ignorando.")
		return
	}

	switch msg.Type {
	case "LIST":
		p.handleList(conn)

	case "REQUEST_FILE":
		p.handleRequestFile(conn, msg)

	case "TRANSFER":
		destPath := filepath.Join("shared", msg.FileName)
		remoteTime := time.Unix(msg.Timestamp, 0)
		if msg.Timestamp == 0 {
			remoteTime = time.Now()
		}
		if info, err := os.Stat(destPath); err == nil {
			if info.ModTime().After(remoteTime) {
				fmt.Printf("⚠️ Archivo local más reciente (%s), se ignora transferencia\n", msg.FileName)
				logger.AppendToLocalLog(logger.Operation{
					Type:      "TIMESTAMP_CONFLICT",
					FileName:  msg.FileName,
					From:      conn.RemoteAddr().String(),
					Timestamp: time.Now().Unix(),
					Message:   "Archivo local más reciente. Transferencia ignorada.",
				})
				return
			}
		}
		if err := os.WriteFile(destPath, msg.Data, 0644); err != nil {
			fmt.Printf("❌ Error al guardar archivo %s: %v\n", msg.FileName, err)
			return
		}
		fmt.Printf("📥 Archivo %s recibido y guardado\n", msg.FileName)
		logger.AppendToLocalLog(logger.Operation{
			Type:      "TRANSFER",
			FileName:  msg.FileName,
			From:      conn.RemoteAddr().String(),
			Timestamp: time.Now().Unix(),
			Message:   "Archivo recibido exitosamente vía TRANSFER",
		})

	default:
		fmt.Println("⚠️ Tipo de mensaje no reconocido:", msg.Type)
	}
}


func (p *Peer) SendFile(filePath, addr string) error {
	const maxRetries = 3
	const timeout = 5 * time.Second

	if p.ID == 0 {
		return fmt.Errorf("nodo sin ID asignado")
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("dirección inválida: %s", addr)
	}
	peerInfo := PeerInfo{IP: parts[0], Port: parts[1]}

	if !CheckPeerAlive(peerInfo) {
		logger.AppendToLocalLog(logger.Operation{
			Type:      "PEER_UNAVAILABLE",
			FileName:  filepath.Base(filePath),
			From:      GetLocalIP() + ":" + p.Port,
			Timestamp: time.Now().Unix(),
			Message:   fmt.Sprintf("Peer %s:%s no responde", peerInfo.IP, peerInfo.Port),
		})
		return fmt.Errorf("peer %s:%s no disponible", peerInfo.IP, peerInfo.Port)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("no se pudo acceder al archivo: %v", err)
	}

	originalPath := filePath
	filename := filepath.Base(filePath)

	if info.IsDir() {
		tmpZip := filepath.Join(os.TempDir(), info.Name()+".zip")
		if err := utils.ZipFolder(filePath, tmpZip); err != nil {
			return fmt.Errorf("error al comprimir: %v", err)
		}
		filePath = tmpZip
		defer os.Remove(tmpZip)
		filename = info.Name() + ".zip"
	}

	hash, err := utils.CalculateSHA256(filePath)
	if err != nil {
		return fmt.Errorf("error hash: %v", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("🔁 Intento %d para enviar %s...\n", attempt, filename)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		cancel()

		if err != nil {
			lastErr = err
			logger.AppendToLocalLog(logger.Operation{
				Type:      "SEND_FAIL",
				FileName:  filename,
				From:      GetLocalIP() + ":" + p.Port,
				Timestamp: time.Now().Unix(),
				Message:   fmt.Sprintf("Falló intento %d: %v", attempt, err),
			})
			time.Sleep(time.Second * time.Duration(attempt))
			continue
		}
		defer conn.Close()

		file, err := os.Open(filePath)
		if err != nil {
			lastErr = err
			break
		}

		_, err = fmt.Fprintf(conn, filename+"\n"+hash+"\n")
		if err != nil {
			lastErr = err
			file.Close()
			continue
		}

		_, err = io.Copy(conn, file)
		file.Close()
		if err != nil {
			lastErr = err
			continue
		}

		logger.AppendToLocalLog(logger.Operation{
			Type:      "TRANSFER",
			FileName:  filename,
			From:      GetLocalIP() + ":" + p.Port,
			Timestamp: time.Now().Unix(),
			Message:   fmt.Sprintf("Enviado con éxito a %s", addr),
		})
		fmt.Printf("📤 %s enviado exitosamente\n", filename)
		return nil
	}

	// Todos los intentos fallaron: registrar y reintentar más tarde
	logger.AppendToLocalLog(logger.Operation{
		Type:      "SEND_FAIL",
		FileName:  filename,
		From:      GetLocalIP() + ":" + p.Port,
		Timestamp: time.Now().Unix(),
		Message:   fmt.Sprintf("Falló tras %d intentos. Último error: %v", maxRetries, lastErr),
	})

	// Agregar a la cola de reintentos con estructura correcta
	state.AddPendingTask(state.PendingTask{
		Type:      "TRANSFER",
		FileName:  originalPath,
		From:      GetLocalIP() + ":" + p.Port,
		To:        addr,
		Retries:   maxRetries,
		Timestamp: time.Now().Unix(),
	})

	return fmt.Errorf("falló el envío tras %d intentos: %v", maxRetries, lastErr)
}

func (p *Peer) handleRequestFile(conn net.Conn, msg message.Message) {
	path := filepath.Join("shared", msg.FileName)
	f, err := os.Open(path)
	if err != nil {
		logger.AppendToLocalLog(logger.Operation{
			Type:      "REQUEST_FAIL",
			FileName:  msg.FileName,
			From:      conn.RemoteAddr().String(),
			Timestamp: time.Now().Unix(),
			Message:   "Archivo no encontrado",
		})
		resp := message.Message{
			Type:     "ERROR",
			From:     strconv.Itoa(p.ID),
			FileName: msg.FileName,
			Data:     []byte("no se pudo abrir el archivo"),
		}
		data, _ := json.Marshal(resp)
		conn.Write(data)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("❌ Error al leer archivo:", err)
		return
	}

	_, err = utils.CalculateSHA256(path)
	if err != nil {
		fmt.Println("❌ Error al calcular hash:", err)
	}

	resp := message.Message{
		Type:      "TRANSFER",
		From:      strconv.Itoa(p.ID),
		FileName:  msg.FileName,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	packet, _ := json.Marshal(resp)
	conn.Write(packet)

	logger.AppendToLocalLog(logger.Operation{
		Type:      "REQUEST_TRANSFER",
		FileName:  msg.FileName,
		From:      conn.RemoteAddr().String(),
		Timestamp: time.Now().Unix(),
		Message:   "Archivo enviado por solicitud remota",
	})
}

func (p *Peer) RequestRemoteFile(fileName, addr string) error {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("dirección inválida: %s", addr)
	}
	peerInfo := PeerInfo{IP: parts[0], Port: parts[1]}

	if !CheckPeerAlive(peerInfo) {
		logger.AppendToLocalLog(logger.Operation{
			Type:      "PEER_UNAVAILABLE",
			FileName:  fileName,
			From:      GetLocalIP() + ":" + p.Port,
			Timestamp: time.Now().Unix(),
			Message:   fmt.Sprintf("Peer %s:%s no responde", peerInfo.IP, peerInfo.Port),
		})
		return fmt.Errorf("peer %s:%s no disponible", peerInfo.IP, peerInfo.Port)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("error de conexión: %v", err)
	}
	defer conn.Close()

	msg := message.Message{
		Type:     "REQUEST_FILE",
		From:     strconv.Itoa(p.ID),
		FileName: fileName,
	}
	data, _ := json.Marshal(msg)
	conn.Write(data)

	response, err := io.ReadAll(conn)
	if err != nil {
		return fmt.Errorf("error al recibir respuesta: %v", err)
	}

	var resp message.Message
	if err := json.Unmarshal(response, &resp); err != nil {
		return fmt.Errorf("respuesta no válida: %v", err)
	}

	if resp.Type != "TRANSFER" || len(resp.Data) == 0 {
		return fmt.Errorf("respuesta inválida o archivo vacío")
	}

	dest := filepath.Join("shared", "recibido-"+fileName)
	if info, err := os.Stat(dest); err == nil && resp.Timestamp > 0 {
		if info.ModTime().After(time.Unix(resp.Timestamp, 0)) {
			logger.AppendToLocalLog(logger.Operation{
				Type:      "TIMESTAMP_CONFLICT",
				FileName:  fileName,
				From:      addr,
				Timestamp: time.Now().Unix(),
				Message:   "Archivo local más reciente. Descarga omitida.",
			})
			return nil
		}
	}

	if err := os.WriteFile(dest, resp.Data, 0644); err != nil {
		return fmt.Errorf("error al guardar archivo: %v", err)
	}

	logger.AppendToLocalLog(logger.Operation{
		Type:      "REQUEST_RECV",
		FileName:  fileName,
		From:      addr,
		Timestamp: time.Now().Unix(),
		Message:   fmt.Sprintf("Archivo recibido desde %s", addr),
	})
	fmt.Printf("✅ Archivo %s recibido desde %s\n", fileName, addr)

	if resp.Timestamp > 0 {
		modTime := time.Unix(resp.Timestamp, 0)
		entries := state.FileCache[peerInfo.IP]
		found := false
		for i, f := range entries {
			if f.Name == fileName {
				entries[i].ModTime = modTime
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, state.FileInfo{
				Name:    fileName,
				ModTime: modTime,
			})
		}
		state.FileCache[peerInfo.IP] = entries
		state.SaveState()
	}

	return nil
}

func (p *Peer) RetryWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		state.LoadState() // 🧠 Asegura que tienes la última versión del estado

		tasks := state.RetryQueue
		if len(tasks) == 0 {
			continue
		}

		fmt.Printf("🔁 Reintentando %d tarea(s) fallidas...\n", len(tasks))

		var updated []state.PendingTask
		for _, task := range tasks {
			if task.Type != "TRANSFER" {
				updated = append(updated, task)
				continue
			}

			info, err := os.Stat(task.FileName)
			if os.IsNotExist(err) {
				logger.AppendToLocalLog(logger.Operation{
					Type:      "RETRY_SKIPPED",
					FileName:  filepath.Base(task.FileName),
					From:      GetLocalIP() + ":" + p.Port,
					Timestamp: time.Now().Unix(),
					Message:   "Archivo eliminado. Reintento omitido.",
				})
				continue
			}

			if info.ModTime().Unix() > task.Timestamp {
				logger.AppendToLocalLog(logger.Operation{
					Type:      "RETRY_SKIPPED",
					FileName:  filepath.Base(task.FileName),
					From:      GetLocalIP() + ":" + p.Port,
					Timestamp: time.Now().Unix(),
					Message:   "Archivo modificado tras el fallo. Reintento omitido.",
				})
				continue
			}

			if err := p.SendFile(task.FileName, task.To); err != nil {
				task.Retries++
				updated = append(updated, task)
			}
		}

		state.RetryQueue = updated
		state.SaveState()
	}
}

func (p *Peer) SyncWithPeer(peerInfo PeerInfo) {
	addr := net.JoinHostPort(peerInfo.IP, peerInfo.Port)
	fmt.Printf("🔁 Sincronizando con %s...\n", addr)

	remoteTree, err := p.RequestFileTree(addr)
	if err != nil || remoteTree == nil {
		fmt.Printf("❌ No se pudo obtener árbol remoto: %v\n", err)
		return
	}

	remoteFiles := fs.FlattenTree(*remoteTree)
	remoteMap := make(map[string]time.Time)
	for _, f := range remoteFiles {
		remoteMap[f.Name] = f.ModTime
	}

	cached := state.FileCache[peerInfo.IP]
	cacheMap := make(map[string]time.Time)
	for _, f := range cached {
		cacheMap[f.Name] = f.ModTime
	}

	localFiles, err := fs.ListFiles("shared")
	if err != nil {
		fmt.Println("❌ Error al listar archivos locales:", err)
		return
	}
	localMap := make(map[string]time.Time)
	for _, f := range localFiles {
		localMap[f.Name] = f.ModTime
	}

	for name, remoteTime := range remoteMap {
		cachedTime, seen := cacheMap[name]
		if !seen || remoteTime.After(cachedTime) {
			fmt.Printf("📥 Descargando archivo actualizado: %s\n", name)
			if err := p.RequestRemoteFile(name, addr); err != nil {
				fmt.Printf("⚠️ Fallo al sincronizar %s: %v\n", name, err)
				continue
			}
			logger.AppendToLocalLog(logger.Operation{
				Type:      "SYNC_FILE",
				FileName:  name,
				From:      addr,
				Timestamp: time.Now().Unix(),
				Message:   "Archivo sincronizado tras reconexión",
			})
			cacheMap[name] = remoteTime
		}
	}

	var updated []state.FileInfo
	for name, mod := range cacheMap {
		updated = append(updated, state.FileInfo{
			Name:    name,
			ModTime: mod,
		})
	}
	state.FileCache[peerInfo.IP] = updated
	state.SaveState()
	fmt.Printf("✅ Sincronización completa con %s\n", addr)
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

