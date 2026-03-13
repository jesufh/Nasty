package render

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// Renderer encapsula la lógica de dibujo en la terminal.
type Renderer struct {
	Screen tcell.Screen
}

// NewRenderer inicializa la pantalla de la terminal.
func NewRenderer() (*Renderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("error creando screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("error inicializando screen: %w", err)
	}

	screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset))
	screen.EnableMouse()
	screen.EnablePaste()
	screen.Clear()

	return &Renderer{
		Screen: screen,
	}, nil
}

// Close restaura la terminal a su estado original.
func (r *Renderer) Close() {
	r.Screen.Fini()
}

// DrawText escribe texto en el grid manejando correctamente UTF-8.
func (r *Renderer) DrawText(x, y int, style tcell.Style, text string) {
	col := 0
	for _, ch := range text {
		r.Screen.SetContent(x+col, y, ch, nil, style)
		// Calculamos cuánto ancho visual ocupa el carácter (normalmente 1, pero emojis o CJK son 2)
		w := runewidth.RuneWidth(ch)
		if w == 0 {
			w = 1 // Fallback seguro
		}
		col += w
	}
}

// Sync obliga a dibujar los cambios en la pantalla.
func (r *Renderer) Sync() {
	r.Screen.Show()
}