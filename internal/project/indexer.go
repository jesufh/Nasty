package project

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Indexer mantiene la lista de todos los archivos del proyecto
type Indexer struct {
	RootPath string
	Files    []string
}

func NewIndexer(root string) *Indexer {
	return &Indexer{
		RootPath: root,
		Files:    make([]string, 0),
	}
}

// Scan recorre el proyecto y construye la lista de archivos indexables
func (i *Indexer) Scan() error {
	i.Files = make([]string, 0) // Limpiar lista anterior

	err := filepath.WalkDir(i.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Ignorar errores de permiso
		}

		// Ignorar directorios ocultos o basura común
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != "." { // .git, .vscode, etc
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == "target" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		// Calcular ruta relativa para mostrar en la lista
		relPath, err := filepath.Rel(i.RootPath, path)
		if err != nil {
			relPath = path
		}
		
		i.Files = append(i.Files, relPath)
		return nil
	})

	return err
}

// FuzzyMatch filtra la lista de archivos basándose en una query simple
// Retorna los archivos que contienen las letras de la query en orden.
func (i *Indexer) FuzzyMatch(query string) []string {
	if query == "" {
		// Si no hay query, devolver los primeros 50 para no saturar
		limit := 50
		if len(i.Files) < limit {
			limit = len(i.Files)
		}
		return i.Files[:limit]
	}

	query = strings.ToLower(query)
	var matches []string
	
	for _, file := range i.Files {
		if simpleFuzzy(file, query) {
			matches = append(matches, file)
		}
		if len(matches) > 50 { // Límite duro para rendimiento en UI
			break
		}
	}
	return matches
}

// Algoritmo simple de coincidencia: caracteres de query deben aparecer en orden
func simpleFuzzy(target, query string) bool {
	target = strings.ToLower(target)
	tIdx := 0
	qIdx := 0
	
	tRunes := []rune(target)
	qRunes := []rune(query)

	for qIdx < len(qRunes) && tIdx < len(tRunes) {
		if tRunes[tIdx] == qRunes[qIdx] {
			qIdx++
		}
		tIdx++
	}
	return qIdx == len(qRunes)
}