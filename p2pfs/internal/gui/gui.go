package gui

import (
	"fmt"
	"image/color"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"p2pfs/internal/fs"
	"p2pfs/internal/peer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var selectedFile string
var localFileListWidget *fyne.Container
var conn *peer.Peer
var fileButtons map[string]*widget.Button
var selfPort string
var peersList []peer.PeerInfo

// Paleta oscura profesional
var backgroundColor = color.RGBA{R: 34, G: 40, B: 49, A: 255}      // #222831
var panelColor = color.RGBA{R: 57, G: 62, B: 70, A: 255}           // #393E46
var borderColor = color.RGBA{R: 0, G: 173, B: 181, A: 255}         // #00ADB5
var selectedColor = color.RGBA{R: 0, G: 122, B: 255, A: 255}       // #007AFF
var textPrimary = color.RGBA{R: 238, G: 238, B: 238, A: 255}       // #EEEEEE
var textSecondary = color.RGBA{R: 200, G: 200, B: 200, A: 255}     // gris claro para fechas

func StartGUI(selfID int, peers []peer.PeerInfo, self *peer.Peer) {
	conn = self
	fileButtons = make(map[string]*widget.Button)
	peersList = peers
	selfPort = self.Port

	a := app.New()
	w := a.NewWindow(fmt.Sprintf("P2PFS - Nodo %d", selfID))
	w.Resize(fyne.NewSize(1200, 800))

	bg := canvas.NewRectangle(backgroundColor)
	grid := container.NewGridWithColumns(2)

	statusLabel := widget.NewLabel("🟢 Sistema iniciado")
	statusLabel.TextStyle.Bold = true

	updateBtn := widget.NewButton("Actualizar", func() {
		updateLocalFiles()
		statusLabel.SetText("✅ Lista actualizada")
	})

	deleteBtn := widget.NewButton("Eliminar seleccionado", func() {
		if selectedFile == "" {
			dialog.ShowInformation("Aviso", "No hay archivo seleccionado", w)
			return
		}
		err := os.Remove("shared/" + selectedFile)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			updateLocalFiles()
			statusLabel.SetText("🗑️ Archivo eliminado: " + selectedFile)
			selectedFile = ""
		}
	})

	transferBtn := widget.NewButton("Transferir archivo", func() {
		if selectedFile == "" {
			dialog.ShowInformation("Aviso", "Seleccione un archivo primero", w)
			return
		}
		msg := ""
		success := 0
		for _, peerInfo := range peersList {
			if peerInfo.Port == selfPort {
				continue
			}
			addr := fmt.Sprintf("%s:%s", peerInfo.IP, peerInfo.Port)
			err := conn.SendFile("shared/"+selectedFile, addr)
			if err != nil {
				msg += fmt.Sprintf("❌ %s: %v\n", addr, err)
			} else {
				msg += fmt.Sprintf("✅ %s: Enviado\n", addr)
				success++
			}
		}
		dialog.ShowInformation("Transferencia", msg, w)
		statusLabel.SetText(fmt.Sprintf("📤 Archivo enviado a %d nodo(s)", success))
	})

	buttonBar := container.NewHBox(updateBtn, deleteBtn, transferBtn)

	for _, p := range peersList {
		isLocal := p.ID == selfID
		titleText := ""
		if isLocal {
			titleText = fmt.Sprintf("Máquina Local (%s:%s)", p.IP, p.Port)
		} else {
			titleText = fmt.Sprintf("Máquina %d (%s:%s)", p.ID, p.IP, p.Port)
		}
		iconStatus := widget.NewIcon(theme.CancelIcon())
		connTest, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", p.IP, p.Port), 500*time.Millisecond)
		if err == nil {
			iconStatus = widget.NewIcon(theme.ConfirmIcon())
			connTest.Close()
		}
		title := container.NewCenter(
			container.NewHBox(
				canvas.NewText(titleText, textPrimary),
				iconStatus,
			),
		)
		var files []fs.FileInfo
		if isLocal {
			localFiles, err := fs.ListFiles("shared")
			if err == nil {
				for _, f := range localFiles {
					if f.Name != "shared" && !strings.HasPrefix(f.Name, "recibido") {
						files = append(files, f)
					}
				}
			}
		} else {
			files = []fs.FileInfo{}
		}
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
		frame := container.NewMax(
			bgPanel,
			container.NewVBox(title, fileList),
		)
		border := canvas.NewRectangle(borderColor)
		border.SetMinSize(fyne.NewSize(570, 230))
		grid.Add(container.NewMax(border, frame))
	}
	content := container.NewBorder(buttonBar, nil, nil, nil, container.NewVBox(statusLabel, grid))
	w.SetContent(container.NewMax(bg, content))
	w.ShowAndRun()
}

func updateLocalFiles() {
	localFiles, err := fs.ListFiles("shared")
	if err != nil {
		return
	}
	fileButtons = make(map[string]*widget.Button)
	fileRows := []fyne.CanvasObject{}
	for _, f := range localFiles {
		if f.Name != "shared" && !strings.HasPrefix(f.Name, "recibido") {
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
	}
	localFileListWidget.Objects = fileRows
	localFileListWidget.Refresh()
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
