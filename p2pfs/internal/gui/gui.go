// gui.go - GUI con árbol jerárquico con rutas relativas consistentes
package gui

import (
	"fmt"
	"image/color"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"p2pfs/internal/fs"
	"p2pfs/internal/peer"
	"p2pfs/internal/state"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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
var getPeersFunc func() []peer.PeerInfo
var mainPanel *fyne.Container

var backgroundColor = color.RGBA{R: 34, G: 40, B: 49, A: 255}
var panelColor = color.RGBA{R: 57, G: 62, B: 70, A: 255}
var borderColor = color.RGBA{R: 0, G: 173, B: 181, A: 255}
var selectedColor = color.RGBA{R: 0, G: 122, B: 255, A: 255}
var textPrimary = color.RGBA{R: 238, G: 238, B: 238, A: 255}
var textSecondary = color.RGBA{R: 200, G: 200, B: 200, A: 255}

func StartGUI(selfID int, getPeers func() []peer.PeerInfo, self *peer.Peer) {
	conn = self
	fileButtons = make(map[string]*widget.Button)
	getPeersFunc = getPeers
	selfPort = self.Port

	a := app.New()
	w := a.NewWindow(fmt.Sprintf("P2PFS - Nodo %d", selfID))
	w.Resize(fyne.NewSize(1200, 800))

	statusLabel := widget.NewLabel("🟢 Sistema iniciado")
	statusLabel.TextStyle.Bold = true
	mainPanel = container.NewVBox()

	buttonBar := container.NewHBox(
		widget.NewButton("Actualizar", func() {
			refreshUI(w, statusLabel)
			statusLabel.SetText("✅ Refrescado")
		}),
		widget.NewButton("Eliminar seleccionado", func() {
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
		}),
		widget.NewButton("Transferir archivo", func() {
			if selectedFile == "" {
				dialog.ShowInformation("Aviso", "Seleccione un archivo primero", w)
				return
			}
			msg := ""
			success := 0
			for _, peerInfo := range getPeersFunc() {
				if peerInfo.Port == selfPort {
					continue
				}
				addr := fmt.Sprintf("%s:%s", peerInfo.IP, peerInfo.Port)
				err := conn.SendFile("shared/" + selectedFile, addr)
				if err != nil {
					msg += fmt.Sprintf("❌ %s: %v\n", addr, err)
				} else {
					msg += fmt.Sprintf("✅ %s: Enviado\n", addr)
					success++
				}
			}
			dialog.ShowInformation("Transferencia", msg, w)
			statusLabel.SetText(fmt.Sprintf("📤 Archivo enviado a %d nodo(s)", success))
		}),
	)

	content := container.NewBorder(buttonBar, nil, nil, nil, mainPanel)
	w.SetContent(content)

	refreshUI(w, statusLabel)
	go func() {
		for range time.Tick(3 * time.Second) {
			refreshUI(w, statusLabel)
		}
	}()

	w.ShowAndRun()
}

func refreshUI(w fyne.Window, statusLabel *widget.Label) {
	allPeers := getPeersFunc()
	var slots [4]*peer.PeerInfo

	slots[0] = &peer.PeerInfo{
		ID:   conn.ID,
		IP:   conn.IP,
		Port: conn.Port,
	}

	idx := 1
	for _, p := range allPeers {
		if p.IP == conn.IP && p.Port == conn.Port {
			continue
		}
		if idx < 4 {
			slots[idx] = &p
			idx++
		}
	}

	grid := container.NewGridWrap(fyne.NewSize(900, 420))

	for i := 0; i < 4; i++ {
		p := slots[i]

		if p == nil {
			emptyTitle := widget.NewLabelWithStyle(fmt.Sprintf("Espacio vacío #%d", i+1), fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
			box := container.NewVBox(emptyTitle, widget.NewLabel("Nodo no disponible"))
			grid.Add(container.NewVBox(box))
			continue
		}

		isLocal := p.IP == conn.IP && p.Port == conn.Port
		var treeRoot fs.FileNode
		titleText := fmt.Sprintf("Máquina %d (%s:%s)", p.ID, p.IP, p.Port)
		peerAddr := fmt.Sprintf("%s:%s", p.IP, p.Port)

		iconStatus := widget.NewIcon(theme.CancelIcon())
		if connTest, err := net.DialTimeout("tcp", peerAddr, 500*time.Millisecond); err == nil {
			iconStatus = widget.NewIcon(theme.ConfirmIcon())
			connTest.Close()
		}

		if isLocal {
			titleText = fmt.Sprintf("Máquina Local (%s:%s)", p.IP, p.Port)
			treeRoot, _ = fs.BuildFileTree("shared")
			treeRoot.Name = ""
		} else {
			treeRoot = fs.FileNode{}
			if state.OnlineStatus[p.IP] {
				treePtr, err := conn.RequestFileTree(peerAddr)
				if err == nil && treePtr != nil {
					treeRoot = *treePtr
				}
			}
			if treeRoot.Name == "" {
				files := state.FileCache[p.IP]
				treeRoot = fs.FileNode{Name: "/", IsDir: true}
				for _, f := range files {
					treeRoot.Children = append(treeRoot.Children, fs.FileNode{Name: f.Name, IsDir: false, ModTime: f.ModTime})
				}
			}
		}

		idMap := make(map[string]fs.FileNode)
		var collect func(node fs.FileNode, prefix string)
		collect = func(node fs.FileNode, prefix string) {
			var path string
			if prefix == "" {
				path = node.Name
			} else {
				path = filepath.Join(prefix, node.Name)
			}
			idMap[path] = node
			if node.IsDir {
				for _, child := range node.Children {
					collect(child, path)
				}
			}
		}
		collect(treeRoot, "")

		tree := widget.NewTree(
			func(uid string) []string {
				node := idMap[uid]
				children := []string{}
				for _, child := range node.Children {
					children = append(children, filepath.Join(uid, child.Name))
				}
				return children
			},
			func(uid string) bool {
				return idMap[uid].IsDir
			},
			func(branch bool) fyne.CanvasObject {
				return container.NewHBox(widget.NewIcon(theme.DocumentIcon()), widget.NewLabel("Nombre"))
			},
			func(uid string, branch bool, obj fyne.CanvasObject) {
				node := idMap[uid]
				box := obj.(*fyne.Container)
				label := box.Objects[1].(*widget.Label)
				label.SetText(node.Name)
				icon := box.Objects[0].(*widget.Icon)
				if node.IsDir {
					icon.SetResource(theme.FolderIcon())
				} else {
					icon.SetResource(theme.DocumentIcon())
				}
			},
		)

		openAllBranches(tree, treeRoot, "")

		tree.OnSelected = func(uid string) {
			if !idMap[uid].IsDir {
				fileName := uid
				if isLocal {
					selectedFile = fileName
				} else {
					dialog.ShowCustomConfirm("Descargar archivo", "Descargar", "Cancelar", widget.NewLabel(fileName), func(ok bool) {
						if ok {
							conn.RequestRemoteFile(fileName, peerAddr)
						}
					}, w)
				}
			}
		}

		expandBtn := widget.NewButton("Abrir todo", func() {
			openAllBranches(tree, treeRoot, "")
		})
		collapseBtn := widget.NewButton("Cerrar todo", func() {
			collapseAllBranches(tree, treeRoot, "")
		})
		treeControls := container.NewHBox(expandBtn, collapseBtn)

		// 🆕 Scroll interno para múltiples archivos
		scrollTree := container.NewVScroll(tree)
		scrollTree.SetMinSize(fyne.NewSize(540, 300))

		fileList := container.NewVBox(treeControls, scrollTree)
		if isLocal {
			localFileListWidget = fileList
		}

		titleLabel := widget.NewLabelWithStyle(titleText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		title := container.NewHBox(titleLabel, iconStatus)

		frame := container.NewVBox(title, fileList)
		grid.Add(frame)
	}

	mainPanel.Objects = []fyne.CanvasObject{grid}
	mainPanel.Refresh()
}

func collapseAllBranches(tree *widget.Tree, node fs.FileNode, prefix string) {
	var path string
	if prefix == "" {
		path = node.Name
	} else {
		path = filepath.Join(prefix, node.Name)
	}
	tree.CloseBranch(path)
	if node.IsDir {
		for _, child := range node.Children {
			collapseAllBranches(tree, child, path)
		}
	}
}



func openAllBranches(tree *widget.Tree, node fs.FileNode, prefix string) {
	var path string
	if prefix == "" {
		path = node.Name
	} else {
		path = filepath.Join(prefix, node.Name)
	}
	tree.OpenBranch(path)
	if node.IsDir {
		for _, child := range node.Children {
			openAllBranches(tree, child, path)
		}
	}
}

func iconoPorNombre(name string) fyne.Resource {
	switch {
	case strings.HasSuffix(strings.ToLower(name), ".pdf"):
		return theme.FileIcon()
	case strings.HasSuffix(name, ".zip"), strings.HasSuffix(name, ".rar"):
		return theme.FolderIcon()
	default:
		return theme.DocumentIcon()
	}
}

func updateLocalFiles() {
	// Puedes usar esta función para ejecutar acciones después de eliminar archivos locales
}


