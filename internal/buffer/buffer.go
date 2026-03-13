package buffer

import "os"

type ActionType int

const (
	ActionNone ActionType = iota
	ActionInsertText
	ActionInsertSpace
	ActionDelete
)

type Snapshot struct {
	pieces []Piece
	cursor int
}

type Buffer struct {
	pt         *PieceTable
	history    []Snapshot
	historyIdx int
	savedIdx   int
	
	// CAMBIO: Exportado (Mayúscula) para acceso desde editor.go
	CursorOffset int 
	
	targetCol  int
	lastAction ActionType
	FilePath   string

	ScrollY int
	ScrollX int

	// NUEVO: Selección (-1 = inactiva)
	SelectionStart int 

	// Estado del Modo Sandbox
	InSandbox      bool
	sandboxBaseIdx int 
}

func NewBuffer(text, path string) *Buffer {
	b := &Buffer{
		pt:             NewPieceTable(text),
		history:        make([]Snapshot, 0),
		historyIdx:     -1,
		savedIdx:       0,
		lastAction:     ActionNone,
		FilePath:       path,
		ScrollY:        0,
		ScrollX:        0,
		InSandbox:      false,
		SelectionStart: -1, // Inicializar sin selección
	}
	b.commitOrUpdateState(ActionNone)
	b.savedIdx = b.historyIdx
	return b
}

// --- LÓGICA DE SELECCIÓN ---

func (b *Buffer) StartSelection() {
	if b.SelectionStart == -1 {
		b.SelectionStart = b.CursorOffset
	}
}

func (b *Buffer) ClearSelection() {
	b.SelectionStart = -1
}

// GetSelectedRange retorna (inicio, longitud) normalizados
func (b *Buffer) GetSelectedRange() (int, int) {
	if b.SelectionStart == -1 {
		return -1, 0
	}
	start := b.SelectionStart
	end := b.CursorOffset
	if start > end {
		start, end = end, start
	}
	return start, end - start
}

func (b *Buffer) GetSelectedText() string {
	start, length := b.GetSelectedRange()
	if start == -1 || length == 0 {
		return ""
	}
	fullText := []rune(b.String())
	if start+length > len(fullText) {
		length = len(fullText) - start
	}
	return string(fullText[start : start+length])
}

func (b *Buffer) DeleteSelection() {
	start, length := b.GetSelectedRange()
	if start != -1 && length > 0 {
		b.pt.Delete(start, length)
		b.CursorOffset = start
		b.commitOrUpdateState(ActionDelete)
		b.ClearSelection()
	}
}

// --- SANDBOX ---

func (b *Buffer) EnterSandbox() {
	if !b.InSandbox {
		b.breakUndoGroup()
		b.InSandbox = true
		b.sandboxBaseIdx = b.historyIdx
	}
}

func (b *Buffer) ApplySandbox() {
	b.InSandbox = false
}

func (b *Buffer) DiscardSandbox() {
	if b.InSandbox {
		if b.sandboxBaseIdx >= 0 && b.sandboxBaseIdx < len(b.history) {
			b.historyIdx = b.sandboxBaseIdx
			b.restore(b.history[b.historyIdx])
			b.history = b.history[:b.historyIdx+1]
		}
		b.InSandbox = false
	}
}

// --- IO Y ESTADO ---

func (b *Buffer) Save() error {
	if b.FilePath == "" {
		return nil
	}
	err := os.WriteFile(b.FilePath, []byte(b.String()), 0644)
	if err == nil {
		b.savedIdx = b.historyIdx
	}
	return err
}

func (b *Buffer) IsModified() bool {
	return b.historyIdx != b.savedIdx
}

func (b *Buffer) commitOrUpdateState(action ActionType) {
	if b.historyIdx < len(b.history)-1 {
		b.history = b.history[:b.historyIdx+1]
	}

	snapshotPieces := make([]Piece, len(b.pt.pieces))
	copy(snapshotPieces, b.pt.pieces)

	newSnap := Snapshot{
		pieces: snapshotPieces,
		cursor: b.CursorOffset,
	}

	if action == b.lastAction && action != ActionNone {
		b.history[b.historyIdx] = newSnap
	} else {
		b.history = append(b.history, newSnap)
		b.historyIdx++
		b.lastAction = action
	}
}

func (b *Buffer) breakUndoGroup() {
	b.lastAction = ActionNone
}

func (b *Buffer) restore(snap Snapshot) {
	b.pt.pieces = make([]Piece, len(snap.pieces))
	copy(b.pt.pieces, snap.pieces)
	b.CursorOffset = snap.cursor
	b.pt.recalculateLength()
	b.updateTargetCol()
}

func (b *Buffer) Undo() {
	b.breakUndoGroup()
	if b.historyIdx > 0 {
		b.historyIdx--
		b.restore(b.history[b.historyIdx])
	}
}

func (b *Buffer) Redo() {
	b.breakUndoGroup()
	if b.historyIdx < len(b.history)-1 {
		b.historyIdx++
		b.restore(b.history[b.historyIdx])
	}
}

// --- MOVIMIENTO ---

func (b *Buffer) updateTargetCol() {
	_, col := b.CursorPos()
	b.targetCol = col
}

func (b *Buffer) Insert(text string) {
	action := ActionInsertText
	if text == " " || text == "\n" || text == "\t" {
		action = ActionInsertSpace
	}

	b.pt.Insert(b.CursorOffset, text)
	b.CursorOffset += len([]rune(text))
	b.updateTargetCol()

	b.commitOrUpdateState(action)
}

func (b *Buffer) DeleteBackwards() {
	if b.CursorOffset > 0 {
		b.pt.Delete(b.CursorOffset-1, 1)
		b.CursorOffset--
		b.updateTargetCol()
		b.commitOrUpdateState(ActionDelete)
	}
}

func (b *Buffer) MoveLeft() {
	b.breakUndoGroup()
	if b.CursorOffset > 0 {
		b.CursorOffset--
		b.updateTargetCol()
	}
}

func (b *Buffer) MoveRight() {
	b.breakUndoGroup()
	if b.CursorOffset < len([]rune(b.String())) {
		b.CursorOffset++
		b.updateTargetCol()
	}
}

func (b *Buffer) MoveUp() {
	b.breakUndoGroup()
	row, _ := b.CursorPos()
	if row > 0 {
		b.CursorOffset = b.rowColToOffset(row-1, b.targetCol)
	} else {
		b.CursorOffset = 0
	}
}

func (b *Buffer) MoveDown() {
	b.breakUndoGroup()
	row, _ := b.CursorPos()
	b.CursorOffset = b.rowColToOffset(row+1, b.targetCol)
}

func (b *Buffer) CursorPos() (int, int) {
	runes := []rune(b.String())
	row, col := 0, 0
	for i := 0; i < b.CursorOffset && i < len(runes); i++ {
		if runes[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return row, col
}

// OffsetOfLine retorna el índice global donde comienza una línea (para renderizado optimizado)
func (b *Buffer) OffsetOfLine(lineIdx int) int {
	runes := []rune(b.String())
	currentLine := 0
	for i, r := range runes {
		if currentLine == lineIdx {
			return i
		}
		if r == '\n' {
			currentLine++
		}
	}
	if currentLine == lineIdx {
		return len(runes) // Fin del archivo
	}
	return -1 // Línea no existe
}

func (b *Buffer) rowColToOffset(targetRow, targetCol int) int {
	runes := []rune(b.String())
	row, col, offset := 0, 0, 0
	for i := 0; i < len(runes); i++ {
		if row == targetRow && (col == targetCol || runes[i] == '\n') {
			return offset
		}
		if runes[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
		offset++
	}
	return offset
}

func (b *Buffer) String() string {
	return b.pt.String()
}

func (b *Buffer) Cursor() int {
	return b.CursorOffset
}

func (b *Buffer) DeleteRange(offset, length int) {
	if offset < 0 || length <= 0 || offset+length > len([]rune(b.String())) {
		return
	}
	b.pt.Delete(offset, length)
	b.commitOrUpdateState(ActionDelete)
}

func (b *Buffer) FindNext(query string, offsetFromCursor int) bool {
	if query == "" {
		return false
	}
	runes := []rune(b.String())
	qRunes := []rune(query)
	qLen := len(qRunes)

	start := b.CursorOffset + offsetFromCursor
	if start > len(runes) {
		start = 0
	}

	for i := start; i <= len(runes)-qLen; i++ {
		match := true
		for j := 0; j < qLen; j++ {
			if runes[i+j] != qRunes[j] {
				match = false
				break
			}
		}
		if match {
			b.CursorOffset = i
			b.updateTargetCol()
			return true
		}
	}

	// Wrap around
	for i := 0; i < start && i <= len(runes)-qLen; i++ {
		match := true
		for j := 0; j < qLen; j++ {
			if runes[i+j] != qRunes[j] {
				match = false
				break
			}
		}
		if match {
			b.CursorOffset = i
			b.updateTargetCol()
			return true
		}
	}

	return false
}