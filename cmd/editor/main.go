// cmd/editor/main.go
package main

import (
	"fmt"
	"os"

	"nasty/internal/core"
	"nasty/internal/render"
)

func main() {
	// 1. Inicializar subsistema de renderizado
	renderer, err := render.NewRenderer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fatal al iniciar la interfaz: %v\n", err)
		os.Exit(1)
	}
	// Asegurarnos de limpiar la terminal pase lo que pase
	defer renderer.Close()

	// 2. Inicializar el Core del editor
	editor := core.NewEditor(renderer)

	// 3. Arrancar el loop principal (Bloqueante)
	editor.Run()
}