package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"nasty/internal/buffer"
	"nasty/internal/project"
	"nasty/internal/render"
	"nasty/internal/syntax"
)

type FocusArea int
type EditorMode int

const (
	FocusEditor FocusArea = iota
	FocusExplorer
)

const (
	ModeNormal EditorMode = iota
	ModeSearch
	ModeReplaceFind
	ModeReplaceWith
	ModeReplaceAsk
	ModeSandboxAsk
	ModeVisual
	ModePalette    
	ModeFileFinder 
)

var internalClipboard string

type Editor struct {
	renderer     *render.Renderer
	buffers      []*buffer.Buffer
	activeBuf    int
	
	// NUEVO: Control de scroll de pestañas
	tabOffset    int 

	explorer     *project.Explorer
	indexer      *project.Indexer
	
	shouldQuit   bool 
	
	focus        FocusArea
	statusMsg    string
	mode         EditorMode
	searchQuery  string
	replaceQuery string

	commands        map[string]Command
	
	paletteQuery    string
	paletteFiltered []string
	paletteSel      int
}

func NewEditor(r *render.Renderer) *Editor {
	cwd, _ := os.Getwd()
	b := buffer.NewBuffer("Bienvenido a Nasty.\nCtrl+K: Comandos | Ctrl+O: Abrir | Ctrl+B: Sandbox", "")

	idx := project.NewIndexer(cwd)
	idx.Scan()

	e := &Editor{
		renderer:  r,
		buffers:   []*buffer.Buffer{b},
		activeBuf: 0,
		tabOffset: 0, // Inicia al principio
		explorer:  project.NewExplorer(cwd),
		indexer:   idx,
		shouldQuit: false,
		focus:     FocusExplorer,
		mode:      ModeNormal,
	}
	
	e.initCommands()
	return e
}

func (e *Editor) currentBuffer() *buffer.Buffer {
	if len(e.buffers) == 0 {
		return nil
	}
	return e.buffers[e.activeBuf]
}

func (e *Editor) Run() {
	e.draw()

	for {
		if e.shouldQuit {
			return
		}

		ev := e.renderer.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			e.renderer.Screen.Sync()
			e.draw()

		case *tcell.EventMouse:
			e.handleMouseEvent(ev)
			e.draw()

		case *tcell.EventKey:
			e.statusMsg = ""
			buf := e.currentBuffer()

			if e.mode == ModePalette || e.mode == ModeFileFinder {
				e.handleModalEvent(ev)
				if e.shouldQuit { return }
				e.draw()
				continue
			}

			if e.mode == ModeVisual {
				e.handleVisualMode(ev)
				e.draw()
				continue
			}

			if e.mode != ModeNormal && buf != nil {
				e.handleModeEvent(ev, buf)
				if e.shouldQuit { return }
				e.draw()
				continue
			}

			switch ev.Key() {
			case tcell.KeyCtrlK: 
				e.mode = ModePalette
				e.paletteQuery = ""
				e.updatePaletteFilter()

			// NUEVO ATAJO: Buscador de archivos
			case tcell.KeyCtrlO: 
				if cmd, ok := e.commands["core.open_file"]; ok { cmd.Action(e) }

			case tcell.KeyCtrlC:
				e.shouldQuit = true
				return

			case tcell.KeyEscape:
				e.mode = ModeNormal
				if buf != nil { buf.ClearSelection() }

			case tcell.KeyTab:
				if e.focus == FocusExplorer { e.focus = FocusEditor } else { e.focus = FocusExplorer }

			case tcell.KeyCtrlS:
				if cmd, ok := e.commands["core.save"]; ok { cmd.Action(e) }
			case tcell.KeyCtrlQ:
				if cmd, ok := e.commands["edit.visual_mode"]; ok { cmd.Action(e) }
			case tcell.KeyCtrlF:
				if cmd, ok := e.commands["edit.find"]; ok { cmd.Action(e) }
			case tcell.KeyCtrlB:
				if cmd, ok := e.commands["tools.sandbox"]; ok { cmd.Action(e) }
			
			// RESTAURADO: Navegación de tabs clásica
			case tcell.KeyCtrlN:
				if cmd, ok := e.commands["buffer.next"]; ok { cmd.Action(e) }
			case tcell.KeyCtrlP:
				if cmd, ok := e.commands["buffer.prev"]; ok { cmd.Action(e) }
			
			case tcell.KeyCtrlW:
				if cmd, ok := e.commands["buffer.close"]; ok { cmd.Action(e) }
			case tcell.KeyCtrlZ:
				if cmd, ok := e.commands["edit.undo"]; ok { cmd.Action(e) }
			case tcell.KeyCtrlY:
				if cmd, ok := e.commands["edit.redo"]; ok { cmd.Action(e) }
			
			case tcell.KeyCtrlV:
				if e.focus == FocusEditor && buf != nil && internalClipboard != "" {
					buf.Insert(internalClipboard)
				}

			case tcell.KeyEnter:
				if e.focus == FocusExplorer {
					e.handleExplorerEnter()
				} else if buf != nil {
					buf.Insert("\n")
				}
			case tcell.KeyUp:
				if e.focus == FocusExplorer { e.explorer.MoveUp() } else if buf != nil { buf.MoveUp() }
			case tcell.KeyDown:
				if e.focus == FocusExplorer { e.explorer.MoveDown() } else if buf != nil { buf.MoveDown() }
			case tcell.KeyLeft:
				if e.focus == FocusEditor && buf != nil { buf.MoveLeft() }
			case tcell.KeyRight:
				if e.focus == FocusEditor && buf != nil { buf.MoveRight() }
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if e.focus == FocusEditor && buf != nil { buf.DeleteBackwards() }
			default:
				if e.focus == FocusEditor && ev.Rune() != 0 && buf != nil {
					buf.Insert(string(ev.Rune()))
				}
			}
			
			if e.shouldQuit {
				return
			}
			e.draw()
		}
	}
}

func (e *Editor) openFile(path string) {
	for i, b := range e.buffers {
		if b.FilePath == path {
			e.activeBuf = i
			e.focus = FocusEditor
			return
		}
	}
	data, err := os.ReadFile(path)
	if err == nil {
		newBuf := buffer.NewBuffer(string(data), path)
		e.buffers = append(e.buffers, newBuf)
		e.activeBuf = len(e.buffers) - 1
		e.focus = FocusEditor
		e.statusMsg = "Abierto: " + filepath.Base(path)
	} else {
		e.statusMsg = "Error al abrir: " + err.Error()
	}
}

func (e *Editor) handleExplorerEnter() {
	node := e.explorer.CurrentNode()
	if node != nil {
		if node.IsDir {
			e.explorer.Cwd = node.Path
			e.explorer.Selected = 0
			e.explorer.Refresh()
		} else {
			e.openFile(node.Path)
		}
	}
}

func (e *Editor) handleModalEvent(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.mode = ModeNormal
		e.statusMsg = ""

	case tcell.KeyEnter:
		if e.mode == ModePalette {
			e.executePaletteCommand()
		} else if e.mode == ModeFileFinder {
			if len(e.paletteFiltered) > 0 && e.paletteSel < len(e.paletteFiltered) {
				selectedFile := e.paletteFiltered[e.paletteSel]
				fullPath := filepath.Join(e.explorer.Cwd, selectedFile)
				if e.indexer.RootPath != "" {
					fullPath = filepath.Join(e.indexer.RootPath, selectedFile)
				}
				e.openFile(fullPath)
				e.mode = ModeNormal
			}
		}

	case tcell.KeyUp:
		if e.paletteSel > 0 {
			e.paletteSel--
		}
	case tcell.KeyDown:
		if e.paletteSel < len(e.paletteFiltered)-1 {
			e.paletteSel++
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.paletteQuery) > 0 {
			runes := []rune(e.paletteQuery)
			e.paletteQuery = string(runes[:len(runes)-1])
			if e.mode == ModePalette { e.updatePaletteFilter() } else { e.updateFileFilter() }
		}
	default:
		if ev.Rune() != 0 {
			e.paletteQuery += string(ev.Rune())
			if e.mode == ModePalette { e.updatePaletteFilter() } else { e.updateFileFilter() }
		}
	}
}

func (e *Editor) handleVisualMode(ev *tcell.EventKey) {
	buf := e.currentBuffer()
	if buf == nil { e.mode = ModeNormal; return }
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlQ:
		e.mode = ModeNormal; buf.ClearSelection(); e.statusMsg = ""
	case tcell.KeyCtrlC:
		text := buf.GetSelectedText()
		if text != "" { internalClipboard = text; e.statusMsg = fmt.Sprintf("Copiado %d caracteres", len(text)) }
		e.mode = ModeNormal; buf.ClearSelection()
	case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
		buf.DeleteSelection(); e.mode = ModeNormal; e.statusMsg = "Selección borrada"
	case tcell.KeyLeft: buf.MoveLeft()
	case tcell.KeyRight: buf.MoveRight()
	case tcell.KeyUp: buf.MoveUp()
	case tcell.KeyDown: buf.MoveDown()
	}
}

func (e *Editor) handleMouseEvent(ev *tcell.EventMouse) {
	x, y := ev.Position()
	buttons := ev.Buttons()
	if buttons&tcell.Button1 != 0 {
		width, _ := e.renderer.Screen.Size()
		treeWidth := 30
		if treeWidth >= width { treeWidth = width / 2 }
		if x < treeWidth && y > 0 {
			e.focus = FocusExplorer; listIndex := y - 1
			if listIndex >= 0 && listIndex < len(e.explorer.Nodes) { e.explorer.Selected = listIndex }
		} else if x > treeWidth && y > 1 { e.focus = FocusEditor }
	}
	if buttons&tcell.WheelUp != 0 {
		if e.focus == FocusEditor && e.currentBuffer() != nil {
			if e.currentBuffer().ScrollY > 0 { e.currentBuffer().ScrollY-- }
		} else if e.focus == FocusExplorer { e.explorer.MoveUp() }
	}
	if buttons&tcell.WheelDown != 0 {
		if e.focus == FocusEditor && e.currentBuffer() != nil { e.currentBuffer().ScrollY++ } else if e.focus == FocusExplorer { e.explorer.MoveDown() }
	}
}

func (e *Editor) handleModeEvent(ev *tcell.EventKey, buf *buffer.Buffer) {
	if ev.Key() == tcell.KeyEscape { e.mode = ModeNormal; e.statusMsg = "Cancelado."; return }
	switch e.mode {
	case ModeSearch:
		if ev.Key() == tcell.KeyEnter {
			if buf.FindNext(e.searchQuery, 1) { e.statusMsg = "Encontrado" } else { e.statusMsg = "Fin" }
		} else { e.updateQueryString(&e.searchQuery, ev) }
	case ModeReplaceFind:
		if ev.Key() == tcell.KeyEnter { if e.searchQuery != "" { e.mode = ModeReplaceWith } } else { e.updateQueryString(&e.searchQuery, ev) }
	case ModeReplaceWith:
		if ev.Key() == tcell.KeyEnter {
			if buf.FindNext(e.searchQuery, 0) { e.mode = ModeReplaceAsk } else { e.mode = ModeNormal; e.statusMsg = "No encontrado." }
		} else { e.updateQueryString(&e.replaceQuery, ev) }
	case ModeReplaceAsk:
		switch ev.Rune() {
		case 's', 'S':
			buf.DeleteRange(buf.Cursor(), len([]rune(e.searchQuery))); buf.Insert(e.replaceQuery)
			if !buf.FindNext(e.searchQuery, 0) { e.mode = ModeNormal }
		case 'n', 'N':
			if !buf.FindNext(e.searchQuery, 1) { e.mode = ModeNormal }
		case 't', 'T':
			count := 0
			for { buf.DeleteRange(buf.Cursor(), len([]rune(e.searchQuery))); buf.Insert(e.replaceQuery); count++; if !buf.FindNext(e.searchQuery, 0) { break } }
			e.mode = ModeNormal; e.statusMsg = fmt.Sprintf("Reemplazados %d.", count)
		case 'c', 'C': e.mode = ModeNormal
		}
	case ModeSandboxAsk:
		switch ev.Rune() {
		case 'a', 'A': buf.ApplySandbox(); e.mode = ModeNormal; e.statusMsg = "Aplicado."
		case 'd', 'D': buf.DiscardSandbox(); e.mode = ModeNormal; e.statusMsg = "Descartado."
		case 'c', 'C': e.mode = ModeNormal
		}
	}
}

func (e *Editor) updateQueryString(query *string, ev *tcell.EventKey) {
	if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
		if len(*query) > 0 { runes := []rune(*query); *query = string(runes[:len(runes)-1]) }
	} else if ev.Rune() != 0 { *query += string(ev.Rune()) }
}

// DRAW: Renderizado principal
func (e *Editor) draw() {
	e.renderer.Screen.Clear()
	width, height := e.renderer.Screen.Size()

	// --- LAYOUT BASICO ---
	treeWidth := 30
	if treeWidth >= width { treeWidth = width / 2 }
	buf := e.currentBuffer()

	// Header
	headerBg := tcell.ColorGreen
	if buf != nil && buf.InSandbox { headerBg = tcell.ColorOlive }
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(headerBg)
	e.renderer.DrawText(0, 0, headerStyle, strings.Repeat(" ", width))
	e.renderer.DrawText(0, 0, headerStyle, " NASTY EDITOR | Ctrl+K: Menú | Ctrl+O: Abrir | Ctrl+W: Cerrar")

	// Explorer
	for i, node := range e.explorer.Nodes {
		y := i + 1
		if y >= height-1 { break }
		style := tcell.StyleDefault
		prefix := "  "
		if node.IsDir { prefix = "▶ "; style = style.Foreground(tcell.ColorBlue).Bold(true) }
		if i == e.explorer.Selected {
			if e.focus == FocusExplorer { style = style.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack) } else { style = style.Background(tcell.ColorDarkGray) }
		}
		text := prefix + node.Name
		if len(text) > treeWidth-1 { text = text[:treeWidth-1] }
		e.renderer.DrawText(0, y, style, text+strings.Repeat(" ", treeWidth-len([]rune(text))))
	}
	for i := 1; i < height-1; i++ { e.renderer.Screen.SetContent(treeWidth, i, '│', nil, tcell.StyleDefault) }

	editorX := treeWidth + 1
	editorWidth := width - editorX

	// --- TABS (Con lógica de Scroll) ---
	tabY := 1
	e.renderer.DrawText(editorX, tabY, tcell.StyleDefault.Background(tcell.ColorDarkGray), strings.Repeat(" ", editorWidth))

	// Lógica de cálculo de offset para scroll de tabs
	// 1. Asegurar que activeBuf no sea menor que el offset
	if e.activeBuf < e.tabOffset {
		e.tabOffset = e.activeBuf
	}

	// 2. Asegurar que activeBuf quepa en la pantalla hacia la derecha
	// Simulamos el renderizado desde tabOffset hasta activeBuf
	for {
		usedWidth := 0
		fits := true
		for i := e.tabOffset; i <= e.activeBuf && i < len(e.buffers); i++ {
			b := e.buffers[i]
			title := filepath.Base(b.FilePath)
			if title == "." || title == "" { title = "untitled" }
			if b.IsModified() { title += "*" }
			if b.InSandbox { title = "[S] " + title }
			// Longitud: espacio + titulo + espacio + separador
			labelLen := len(title) + 3 
			usedWidth += labelLen
			
			if usedWidth > editorWidth {
				fits = false
				break
			}
		}
		
		if fits {
			break
		}
		// Si no cabe, desplazamos el offset a la derecha
		e.tabOffset++
		if e.tabOffset > e.activeBuf {
			e.tabOffset = e.activeBuf // Seguridad
			break
		}
	}

	// 3. Renderizar desde el offset calculado
	currentX := editorX
	// Indicador visual si hay tabs a la izquierda ocultas
	if e.tabOffset > 0 {
		e.renderer.DrawText(currentX, tabY, tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorYellow), "<")
		currentX += 1
	}

	for i := e.tabOffset; i < len(e.buffers); i++ {
		b := e.buffers[i]
		title := filepath.Base(b.FilePath)
		if title == "." || title == "" { title = "untitled" }
		if b.IsModified() { title += "*" }
		if b.InSandbox { title = "[S] " + title }

		style := tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)
		if i == e.activeBuf {
			if b.InSandbox {
				style = tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack).Bold(true)
			} else {
				style = tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack).Bold(true)
			}
		} else if b.InSandbox { style = tcell.StyleDefault.Background(tcell.ColorOlive).Foreground(tcell.ColorWhite) }
		
		label := fmt.Sprintf(" %s ", title)
		
		// Si no cabe el siguiente, paramos (o mostramos indicador de mas)
		if currentX+len(label) > width {
			e.renderer.DrawText(width-1, tabY, tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorYellow), ">")
			break
		}

		e.renderer.DrawText(currentX, tabY, style, label)
		currentX += len(label) + 1 // +1 espacio entre tabs
	}

	// Editor Content
	if buf != nil {
		cursorY, cursorX := buf.CursorPos()
		gutterWidth := 4
		contentX := editorX + gutterWidth + 1
		actualEditorWidth := editorWidth - gutterWidth - 1

		if cursorY < buf.ScrollY { buf.ScrollY = cursorY } else if cursorY >= buf.ScrollY+(height-3) { buf.ScrollY = cursorY - (height - 3) + 1 }
		if cursorX < buf.ScrollX { buf.ScrollX = cursorX } else if cursorX >= buf.ScrollX+actualEditorWidth { buf.ScrollX = cursorX - actualEditorWidth + 1 }

		lines := strings.Split(buf.String(), "\n")
		qRunes := []rune(e.searchQuery)
		qLen := len(qRunes)
		selStart, selLen := buf.GetSelectedRange()
		currentGlobalOffset := buf.OffsetOfLine(buf.ScrollY)

		for i := 0; i < height-3; i++ {
			lineIdx := buf.ScrollY + i
			if lineIdx >= len(lines) { break }

			lineStr := lines[lineIdx]
			runes := []rune(lineStr)
			styles := syntax.HighlightLine(lineStr, buf.FilePath)

			// Gutter
			lineNumStr := fmt.Sprintf("%4d", lineIdx+1)
			lineStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
			if lineIdx == cursorY { lineStyle = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true) }
			e.renderer.DrawText(editorX, i+2, lineStyle, lineNumStr)
			e.renderer.Screen.SetContent(editorX+gutterWidth, i+2, '│', nil, tcell.StyleDefault.Foreground(tcell.ColorDarkGray))

			// Match Search
			matchHighlight := make([]bool, len(runes))
			if e.mode != ModeNormal && e.mode != ModeSandboxAsk && e.mode != ModePalette && e.mode != ModeFileFinder && qLen > 0 {
				for k := 0; k <= len(runes)-qLen; k++ {
					match := true
					for m := 0; m < qLen; m++ { if runes[k+m] != qRunes[m] { match = false; break } }
					if match { for m := 0; m < qLen; m++ { matchHighlight[k+m] = true } }
				}
			}

			// Render Text
			for j := 0; j < actualEditorWidth; j++ {
				runeIdx := buf.ScrollX + j
				if runeIdx < len(runes) {
					style := styles[runeIdx]
					if matchHighlight[runeIdx] { style = style.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack).Bold(true) }
					charGlobalOffset := currentGlobalOffset + runeIdx
					if selStart != -1 && charGlobalOffset >= selStart && charGlobalOffset < selStart+selLen {
						style = style.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
					}
					e.renderer.Screen.SetContent(contentX+j, i+2, runes[runeIdx], nil, style)
				}
			}
			currentGlobalOffset += len(runes) + 1
		}
		if e.focus == FocusEditor && e.mode == ModeNormal {
			e.renderer.Screen.ShowCursor(contentX+(cursorX-buf.ScrollX), (cursorY-buf.ScrollY)+2)
		}
	}

	// Footer
	footerStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue)
	footerText := strings.Repeat(" ", width)
	switch e.mode {
	case ModeSearch:
		footerStyle = footerStyle.Background(tcell.ColorPurple)
		footerText = " BUSCAR: " + e.searchQuery + "█"
	case ModeReplaceFind:
		footerStyle = footerStyle.Background(tcell.ColorPurple)
		footerText = " REEMPLAZAR: " + e.searchQuery + "█"
	case ModeReplaceWith:
		footerStyle = footerStyle.Background(tcell.ColorPurple)
		footerText = " CON: " + e.replaceQuery + "█"
	case ModeReplaceAsk:
		footerStyle = footerStyle.Background(tcell.ColorMaroon)
		footerText = fmt.Sprintf(" ¿REEMPLAZAR '%s' con '%s'? [s/n/t/c]", e.searchQuery, e.replaceQuery)
	case ModeSandboxAsk:
		footerStyle = footerStyle.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack).Bold(true)
		footerText = " SANDBOX: [A]plicar | [D]escartar | [C]ancelar"
	case ModeVisual:
		footerStyle = footerStyle.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite).Bold(true)
		footerText = " -- VISUAL -- | ^C: Copiar | Del: Borrar | Esc"
	case ModePalette:
		footerStyle = footerStyle.Background(tcell.ColorDarkCyan)
		footerText = " COMANDOS: Enter para ejecutar."
	case ModeFileFinder:
		footerStyle = footerStyle.Background(tcell.ColorTeal)
		footerText = " ABRIR ARCHIVO: Escribe para buscar..."
	default:
		fileName := "Ningún archivo"
		if buf != nil && buf.FilePath != "" { fileName = filepath.Base(buf.FilePath) }
		footerText = " " + fileName + " | " + e.statusMsg
	}
	e.renderer.DrawText(0, height-1, footerStyle, strings.Repeat(" ", width))
	e.renderer.DrawText(0, height-1, footerStyle, footerText)

	// --- RENDERIZADO DE MODALES ---
	if e.mode == ModePalette || e.mode == ModeFileFinder {
		title := " COMANDOS "
		color := tcell.ColorDarkCyan
		if e.mode == ModeFileFinder {
			title = " ABRIR ARCHIVO "
			color = tcell.ColorTeal
		}
		e.drawCenteredModal(width, height, title, color, e.paletteQuery, e.paletteFiltered, e.paletteSel)
	}

	if e.focus != FocusEditor || e.mode != ModeNormal {
		e.renderer.Screen.HideCursor()
	}
	e.renderer.Sync()
}

func (e *Editor) drawCenteredModal(screenW, screenH int, title string, color tcell.Color, query string, items []string, selected int) {
	w := screenW / 2
	h := 12
	if w < 50 { w = 50 }
	if w > screenW { w = screenW - 4 }
	
	x := (screenW - w) / 2
	y := screenH / 4 

	boxStyle := tcell.StyleDefault.Background(color).Foreground(tcell.ColorWhite)
	selStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	dimStyle := tcell.StyleDefault.Background(color).Foreground(tcell.ColorLightGray)
	selDimStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorDarkGray)
	
	for i := 0; i < h; i++ {
		e.renderer.DrawText(x, y+i, boxStyle, strings.Repeat(" ", w))
	}
	
	e.renderer.DrawText(x+1, y, boxStyle.Bold(true), "> "+query+"_")
	e.renderer.DrawText(x+w-len(title)-1, y, boxStyle.Bold(true), title)
	
	if w > 2 {
		divider := " " + strings.Repeat("─", w-2) + " "
		e.renderer.DrawText(x, y+1, boxStyle, divider)
	}

	visibleItems := h - 2
	startIdx := 0
	if selected >= visibleItems {
		startIdx = selected - visibleItems + 1
	}

	for i := 0; i < visibleItems; i++ {
		idx := startIdx + i
		if idx >= len(items) {
			break
		}
		
		itemText := items[idx]
		itemDesc := ""
		
		if e.mode == ModePalette {
			if cmd, ok := e.commands[itemText]; ok {
				itemDesc = fmt.Sprintf("(%s)", itemText)
				itemText = cmd.Description
			}
		}

		availableWidth := w - 4
		displayMain := itemText
		displaySec := itemDesc
		
		maxMainLen := availableWidth - len(displaySec) - 1
		if maxMainLen < 5 { maxMainLen = 5 }
		
		if len(displayMain) > maxMainLen {
			displayMain = displayMain[:maxMainLen-1] + "…"
		}
		
		paddingLen := availableWidth - len(displayMain) - len(displaySec)
		if paddingLen < 0 { paddingLen = 0 }
		padding := strings.Repeat(" ", paddingLen)
		
		rowStyle := boxStyle
		secStyle := dimStyle
		if idx == selected {
			rowStyle = selStyle
			secStyle = selDimStyle
		}
		
		e.renderer.DrawText(x+2, y+2+i, rowStyle, displayMain + padding)
		if displaySec != "" {
			e.renderer.DrawText(x+2+len(displayMain)+len(padding), y+2+i, secStyle, displaySec)
		}
	}
}