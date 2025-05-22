package gui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"p2pfs/internal/fs"
	"p2pfs/internal/message"
	"p2pfs/internal/peer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var selectedFile string
var localFileListWidget *fyne.Container
var conn *peer.Peer
var fileButtons map[string]*widget.Button

var backgroundColor = color.RGBA{34, 40, 49, 255}      // #222831
var panelColor = color.RGBA{57, 62, 70, 255}           // #393E46
var borderColor = color.RGBA{0, 173, 181, 255}         // #00ADB5
var selectedColor = color.RGBA{0, 122, 255, 255}       // #007AFF
var textPrimary = color.RGBA{238, 238, 238, 255}       // #EEEEEE
var textSecondary = color.RGBA{200, 200, 200, 255}     // gris claro

func StartGUI(selfID int, peers []peer.PeerInfo, self *peer.Peer) {
	conn = self
	fileButtons = make(map[string]*widget.Button)
	a := app.New()
	w := a.NewWindow(fmt.Sprintf("P2PFS - Nodo %d", selfID))
	w.Resize(fyne.NewSize(1200, 800))

	bg := canvas.NewRectangle(backgroundColor)
	grid := container.NewGridWithColumns(2)

	statusLabel := widget.NewLabel("🟢 Sistema iniciado")
	statusLabel.TextStyle.Bold = true

	updateBtn := widget.NewButton("Actualizar", func() {
		statusLabel.SetText("🔄 Actualizando lista...")
		w.Content().Refresh()
		statusLabel.SetText("✅ Lista actualizada")
	})
	deleteBtn := widget.NewButton("Eliminar seleccionado", func() {
		if selectedFile != "" {
			path := "shared/" + selectedFile
			fs.DeletePath(path)
			statusLabel.SetText("🗑️ Archivo eliminado")
		}
	})
	transferBtn := widget.NewButton("Transferir archivo", func() {
		if selectedFile != "" {
			ruta := "shared/" + selectedFile
			resultados := ""
			for _, p := range peers {
				if p.ID == selfID {
					continue
				}
				err := self.SendFile(ruta, fmt.Sprintf("%s:%s", p.IP, p.Port))
				if err != nil {
					resultados += fmt.Sprintf("❌ %s:%s: %v\n", p.IP, p.Port, err)
				} else {
					resultados += fmt.Sprintf("✅ %s:%s\n", p.IP, p.Port)
				}
			}
			statusLabel.SetText("📤 " + resultados)
		}
	})
	buttonBar := container.NewHBox(updateBtn, deleteBtn, transferBtn)

	for _, p := range peers {
		isLocal := p.ID == selfID
		titleText := ""
		if isLocal {
			titleText = fmt.Sprintf("Máquina Local (%s:%s)", p.IP, p.Port)
		} else {
			titleText = fmt.Sprintf("Máquina %d (%s:%s)", p.ID, p.IP, p.Port)
		}

		iconStatus := widget.NewIcon(theme.CancelIcon())
		addr := fmt.Sprintf("%s:%s", p.IP, p.Port)
		connTest, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		files := []fs.FileInfo{}
		if err == nil {
			iconStatus = widget.NewIcon(theme.ConfirmIcon())
			connTest.Close()
			if isLocal {
				localFiles, err := fs.ListFiles("shared")
				if err == nil {
					files = localFiles
				}
			} else {
				files = obtenerArchivosRemotos(addr, selfID)
			}
		}

		title := container.NewCenter(container.NewHBox(canvas.NewText(titleText, textPrimary), iconStatus))

		fileRows := []fyne.CanvasObject{}
		for _, f := range files {
			name := f.Name
			btn := widget.NewButton(name, nil)
			fileButtons[name] = btn
			icon := widget.NewIcon(iconoPorNombre(name))
			modTime := canvas.NewText(f.ModTime.Format("02/01/2006 15:04"), textSecondary)

			var lastClick time.Time
			btn.OnTapped = func(n string) func() {
				return func() {
					now := time.Now()
					if selectedFile != n {
						selectedFile = n
						for key, b := range fileButtons {
							if key == n {
								b.Importance = widget.HighImportance
							} else {
								b.Importance = widget.MediumImportance
							}
							b.Refresh()
						}
					} else if now.Sub(lastClick) < 500*time.Millisecond {
						_ = exec.Command("xdg-open", "shared/"+n).Start()
					}
					lastClick = now
				}
			}(name)

			fileRows = append(fileRows, container.NewHBox(icon, btn, modTime))
		}

		fileList := container.NewVBox(fileRows...)
		if isLocal {
			localFileListWidget = fileList
		}

		bgPanel := canvas.NewRectangle(panelColor)
		bgPanel.SetMinSize(fyne.NewSize(560, 220))
		frame := container.NewMax(bgPanel, container.NewVBox(title, fileList))
		border := canvas.NewRectangle(borderColor)
		border.SetMinSize(fyne.NewSize(570, 230))
		grid.Add(container.NewMax(border, frame))
	}

	content := container.NewBorder(buttonBar, nil, nil, nil, container.NewVBox(statusLabel, grid))
	w.SetContent(container.NewMax(bg, content))
	w.ShowAndRun()
}

func obtenerArchivosRemotos(addr string, selfID int) []fs.FileInfo {
	var archivos []fs.FileInfo
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return archivos
	}
	defer conn.Close()

	msg := message.Message{
		Type:   "LIST",
		Origin: selfID,
	}
	dat, _ := json.Marshal(msg)
	conn.Write(dat)

	buf := make([]byte, 1<<20)
	n, _ := conn.Read(buf)
	var resp message.Message
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return archivos
	}
	if resp.Tree != nil {
		archivos = extraerArchivosDesdeTree(resp.Tree)
	}
	return archivos
}

func extraerArchivosDesdeTree(nodo *message.FileNode) []fs.FileInfo {
	var archivos []fs.FileInfo
	if nodo == nil {
		return archivos
	}
	if !nodo.IsDir {
		archivos = append(archivos, fs.FileInfo{
			Name:    nodo.Name,
			ModTime: time.Now(),
			IsDir:   false,
		})
	}
	for _, hijo := range nodo.Children {
		archivos = append(archivos, extraerArchivosDesdeTree(&hijo)...)
	}
	return archivos
}

func iconoPorNombre(nombre string) fyne.Resource {
	ext := strings.ToLower(filepath.Ext(nombre))
	switch ext {
	case ".txt", ".log", ".md":
		return theme.DocumentIcon()
	case ".pdf", ".doc", ".docx", ".ppt", ".pptx":
		return theme.DocumentIcon()
	case ".mp3", ".wav":
		return theme.FileAudioIcon()
	case ".mp4", ".avi", ".mov", ".mkv":
		return theme.FileVideoIcon()
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp":
		return theme.FileImageIcon()
	default:
		return theme.FileIcon()
	}
}
