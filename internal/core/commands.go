package core

import (
	"sort"
	"strings"
)

// Command define una acción ejecutable en el editor
type Command struct {
	Name        string
	Description string
	Action      func(e *Editor)
}

// initCommands registra todas las acciones disponibles en el sistema
func (e *Editor) initCommands() {
	e.commands = make(map[string]Command)

	// --- ARCHIVO ---
	e.register("core.open_file", "Abrir archivo (Ctrl+O)", func(e *Editor) {
		go e.indexer.Scan() // Escaneo en background
		e.mode = ModeFileFinder
		e.paletteQuery = ""
		e.updateFileFilter()
		e.statusMsg = "Buscador de archivos (Ctrl+O)"
	})

	e.register("core.save", "Guardar archivo", func(e *Editor) {
		buf := e.currentBuffer()
		if buf != nil {
			if buf.InSandbox {
				e.statusMsg = "SANDBOX BLOQUEADO: Aplica o descarta antes de guardar."
			} else if err := buf.Save(); err != nil {
				e.statusMsg = "Error al guardar: " + err.Error()
			} else {
				e.statusMsg = "Guardado exitosamente: " + buf.FilePath
			}
		}
	})

	e.register("core.quit", "Salir de Nasty", func(e *Editor) {
		e.shouldQuit = true
	})

	// --- PESTAÑAS ---
	e.register("buffer.close", "Cerrar pestaña actual", func(e *Editor) {
		if len(e.buffers) > 0 {
			e.buffers = append(e.buffers[:e.activeBuf], e.buffers[e.activeBuf+1:]...)
			if e.activeBuf >= len(e.buffers) {
				e.activeBuf = len(e.buffers) - 1
			}
			if e.activeBuf < 0 {
				e.activeBuf = 0
			}
			e.statusMsg = "Pestaña cerrada"
		} else {
			e.shouldQuit = true
		}
	})

	e.register("buffer.next", "Ir a la siguiente pestaña", func(e *Editor) {
		if len(e.buffers) > 1 {
			e.activeBuf = (e.activeBuf + 1) % len(e.buffers)
		}
	})

	e.register("buffer.prev", "Ir a la pestaña anterior", func(e *Editor) {
		if len(e.buffers) > 1 {
			e.activeBuf = (e.activeBuf - 1 + len(e.buffers)) % len(e.buffers)
		}
	})

	// --- EDICIÓN ---
	e.register("edit.undo", "Deshacer (Undo)", func(e *Editor) {
		if b := e.currentBuffer(); b != nil {
			b.Undo()
			e.statusMsg = "Deshacer"
		}
	})

	e.register("edit.redo", "Rehacer (Redo)", func(e *Editor) {
		if b := e.currentBuffer(); b != nil {
			b.Redo()
			e.statusMsg = "Rehacer"
		}
	})

	e.register("edit.visual_mode", "Alternar Modo Visual", func(e *Editor) {
		if b := e.currentBuffer(); b != nil {
			e.mode = ModeVisual
			b.StartSelection()
			e.statusMsg = "-- MODO VISUAL --"
		}
	})

	e.register("edit.find", "Buscar en archivo", func(e *Editor) {
		e.mode = ModeSearch
		e.searchQuery = ""
	})

	e.register("edit.replace", "Buscar y Reemplazar", func(e *Editor) {
		e.mode = ModeReplaceFind
		e.searchQuery = ""
		e.replaceQuery = ""
	})

	// --- HERRAMIENTAS ---
	e.register("tools.sandbox", "Activar Sandbox (Snapshot seguro)", func(e *Editor) {
		buf := e.currentBuffer()
		if buf != nil {
			if !buf.InSandbox {
				buf.EnterSandbox()
				e.statusMsg = "MODO SANDBOX ACTIVADO"
			} else {
				e.mode = ModeSandboxAsk
			}
		}
	})

	e.register("view.explorer", "Enfocar Explorador de archivos", func(e *Editor) {
		e.focus = FocusExplorer
	})

	e.register("view.editor", "Enfocar Editor de código", func(e *Editor) {
		e.focus = FocusEditor
	})
}

func (e *Editor) register(name, desc string, action func(e *Editor)) {
	e.commands[name] = Command{Name: name, Description: desc, Action: action}
}

// --- FILTROS ---

func (e *Editor) updatePaletteFilter() {
	query := strings.ToLower(e.paletteQuery)
	var matches []string
	for name, cmd := range e.commands {
		if strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(cmd.Description), query) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	e.paletteFiltered = matches
	e.paletteSel = 0
}

func (e *Editor) updateFileFilter() {
	e.paletteFiltered = e.indexer.FuzzyMatch(e.paletteQuery)
	e.paletteSel = 0
}

func (e *Editor) executePaletteCommand() {
	if len(e.paletteFiltered) > 0 && e.paletteSel < len(e.paletteFiltered) {
		cmdName := e.paletteFiltered[e.paletteSel]
		if cmd, ok := e.commands[cmdName]; ok {
			e.mode = ModeNormal
			cmd.Action(e)
		}
	} else {
		e.mode = ModeNormal
	}
}