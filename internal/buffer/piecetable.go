// internal/buffer/piecetable.go
package buffer

// BufferType identifica de qué banco de memoria proviene el texto
type BufferType int

const (
	Original BufferType = iota
	Add
)

// Piece representa un bloque continuo de texto
type Piece struct {
	Source BufferType
	Start  int // Posición inicial dentro del banco (Original o Add)
	Length int // Cantidad de caracteres (runas)
}

// PieceTable es la estructura central de rendimiento para editar texto
type PieceTable struct {
	original []rune
	add[]rune
	pieces[]Piece
	length   int // Longitud total lógica del archivo
}

// NewPieceTable inicializa la tabla con un texto base
func NewPieceTable(text string) *PieceTable {
	runes :=[]rune(text)
	pt := &PieceTable{
		original: runes,
		add:      []rune{},
		pieces:[]Piece{},
		length:   len(runes),
	}
	if len(runes) > 0 {
		pt.pieces = append(pt.pieces, Piece{Source: Original, Start: 0, Length: len(runes)})
	}
	return pt
}

// String reconstruye el texto completo uniendo las piezas.
func (pt *PieceTable) String() string {
	var result[]rune
	for _, p := range pt.pieces {
		if p.Source == Original {
			result = append(result, pt.original[p.Start:p.Start+p.Length]...)
		} else {
			result = append(result, pt.add[p.Start:p.Start+p.Length]...)
		}
	}
	return string(result)
}

// Insert agrega texto en el offset indicado sin destruir memoria
func (pt *PieceTable) Insert(offset int, text string) {
	newRunes :=[]rune(text)
	if len(newRunes) == 0 {
		return
	}

	addStart := len(pt.add)
	pt.add = append(pt.add, newRunes...)
	newPiece := Piece{Source: Add, Start: addStart, Length: len(newRunes)}

	// Caso: Insertar en archivo vacío
	if offset == 0 && len(pt.pieces) == 0 {
		pt.pieces = append(pt.pieces, newPiece)
		pt.length += len(newRunes)
		return
	}

	currentOffset := 0
	for i, p := range pt.pieces {
		// Encontramos la pieza donde cae el cursor
		if currentOffset+p.Length > offset || (currentOffset+p.Length == offset && i == len(pt.pieces)-1) {
			
			// Si estamos insertando justo al final de la última pieza
			if currentOffset+p.Length == offset {
				pt.pieces = append(pt.pieces, newPiece)
				break
			}

			// Dividimos la pieza actual en dos (Left y Right)
			splitOffset := offset - currentOffset
			leftPiece := Piece{Source: p.Source, Start: p.Start, Length: splitOffset}
			rightPiece := Piece{Source: p.Source, Start: p.Start + splitOffset, Length: p.Length - splitOffset}

			// Construimos el nuevo arreglo de piezas
			newPieces := make([]Piece, 0, len(pt.pieces)+2)
			newPieces = append(newPieces, pt.pieces[:i]...)
			if leftPiece.Length > 0 {
				newPieces = append(newPieces, leftPiece)
			}
			newPieces = append(newPieces, newPiece)
			if rightPiece.Length > 0 {
				newPieces = append(newPieces, rightPiece)
			}
			newPieces = append(newPieces, pt.pieces[i+1:]...)

			pt.pieces = newPieces
			break
		}
		currentOffset += p.Length
	}
	pt.length += len(newRunes)
}

// Delete elimina una cantidad de caracteres de forma lógica (sin borrar memoria real)
func (pt *PieceTable) Delete(offset, length int) {
	if offset < 0 || length <= 0 || offset+length > pt.length {
		return
	}

	newPieces := make([]Piece, 0)
	currentOffset := 0
	deleteStart := offset
	deleteEnd := offset + length

	for _, p := range pt.pieces {
		pieceStart := currentOffset
		pieceEnd := currentOffset + p.Length

		if pieceEnd <= deleteStart {
			newPieces = append(newPieces, p) // Totalmente a la izquierda
		} else if pieceStart >= deleteEnd {
			newPieces = append(newPieces, p) // Totalmente a la derecha
		} else {
			// Superposición: La pieza sobrevive parcialmente
			if pieceStart < deleteStart {
				newPieces = append(newPieces, Piece{Source: p.Source, Start: p.Start, Length: deleteStart - pieceStart})
			}
			if pieceEnd > deleteEnd {
				discarded := deleteEnd - pieceStart
				newPieces = append(newPieces, Piece{Source: p.Source, Start: p.Start + discarded, Length: pieceEnd - deleteEnd})
			}
		}
		currentOffset += p.Length
	}

	pt.pieces = newPieces
	pt.length -= length
}

// recalculateLength es un utilitario para actualizar la longitud tras deshacer cambios
func (pt *PieceTable) recalculateLength() {
	l := 0
	for _, p := range pt.pieces {
		l += p.Length
	}
	pt.length = l
}