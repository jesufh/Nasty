// internal/syntax/highlighter.go
package syntax

import (
	"path/filepath"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

var (
	ColorKeyword = tcell.ColorOrange
	ColorString  = tcell.ColorGreen
	ColorComment = tcell.ColorGray
	ColorNumber  = tcell.ColorAqua
	ColorType    = tcell.ColorYellow
	ColorDefault = tcell.ColorReset
)

// Mapa principal: extensión de archivo -> mapa de palabras clave
var keywords = make(map[string]map[string]tcell.Color)

func init() {
	// 1. Familia C / C++ / C# / Java
	cFamily := map[string]tcell.Color{
		"if": ColorKeyword, "else": ColorKeyword, "for": ColorKeyword, "while": ColorKeyword,
		"return": ColorKeyword, "break": ColorKeyword, "continue": ColorKeyword, "switch": ColorKeyword,
		"case": ColorKeyword, "default": ColorKeyword, "class": ColorKeyword, "struct": ColorKeyword,
		"public": ColorKeyword, "private": ColorKeyword, "protected": ColorKeyword, "static": ColorKeyword,
		"new": ColorKeyword, "try": ColorKeyword, "catch": ColorKeyword, "throw": ColorKeyword,
		"int": ColorType, "char": ColorType, "float": ColorType, "double": ColorType, "void": ColorType,
		"bool": ColorType, "string": ColorType, "true": ColorType, "false": ColorType, "null": ColorType,
	}

	// 2. Familia JS / TS
	jsFamily := map[string]tcell.Color{
		"function": ColorKeyword, "const": ColorKeyword, "let": ColorKeyword, "var": ColorKeyword,
		"return": ColorKeyword, "if": ColorKeyword, "else": ColorKeyword, "for": ColorKeyword,
		"while": ColorKeyword, "class": ColorKeyword, "import": ColorKeyword, "export": ColorKeyword,
		"async": ColorKeyword, "await": ColorKeyword, "try": ColorKeyword, "catch": ColorKeyword,
		"true": ColorType, "false": ColorType, "null": ColorType, "undefined": ColorType,
		"string": ColorType, "number": ColorType, "boolean": ColorType, "any": ColorType,
	}

	// 3. Familia Python
	python := map[string]tcell.Color{
		"def": ColorKeyword, "class": ColorKeyword, "import": ColorKeyword, "from": ColorKeyword,
		"if": ColorKeyword, "elif": ColorKeyword, "else": ColorKeyword, "for": ColorKeyword,
		"while": ColorKeyword, "return": ColorKeyword, "in": ColorKeyword, "and": ColorKeyword,
		"or": ColorKeyword, "not": ColorKeyword, "try": ColorKeyword, "except": ColorKeyword,
		"finally": ColorKeyword, "with": ColorKeyword, "as": ColorKeyword, "pass": ColorKeyword,
		"break": ColorKeyword, "continue": ColorKeyword, "True": ColorType, "False": ColorType, "None": ColorType,
	}

	// 4. Familia Go
	goLang := map[string]tcell.Color{
		"func": ColorKeyword, "package": ColorKeyword, "import": ColorKeyword, "type": ColorKeyword,
		"struct": ColorKeyword, "interface": ColorKeyword, "return": ColorKeyword, "if": ColorKeyword,
		"else": ColorKeyword, "for": ColorKeyword, "switch": ColorKeyword, "case": ColorKeyword,
		"var": ColorKeyword, "const": ColorKeyword, "default": ColorKeyword, "defer": ColorKeyword,
		"go": ColorKeyword, "chan": ColorKeyword, "fallthrough": ColorKeyword, "range": ColorKeyword,
		"string": ColorType, "int": ColorType, "int64": ColorType, "bool": ColorType, "error": ColorType,
		"true": ColorType, "false": ColorType, "nil": ColorType,
	}

	// 5. Rust
	rust := map[string]tcell.Color{
		"fn": ColorKeyword, "let": ColorKeyword, "mut": ColorKeyword, "if": ColorKeyword,
		"else": ColorKeyword, "for": ColorKeyword, "while": ColorKeyword, "loop": ColorKeyword,
		"match": ColorKeyword, "return": ColorKeyword, "struct": ColorKeyword, "enum": ColorKeyword,
		"impl": ColorKeyword, "trait": ColorKeyword, "pub": ColorKeyword, "use": ColorKeyword,
		"i32": ColorType, "i64": ColorType, "f32": ColorType, "f64": ColorType, "bool": ColorType,
		"String": ColorType, "str": ColorType, "true": ColorType, "false": ColorType, "Option": ColorType,
	}

	// 6. Scripts (Bash/Shell)
	shell := map[string]tcell.Color{
		"if": ColorKeyword, "then": ColorKeyword, "else": ColorKeyword, "elif": ColorKeyword, "fi": ColorKeyword,
		"for": ColorKeyword, "while": ColorKeyword, "do": ColorKeyword, "done": ColorKeyword, "case": ColorKeyword,
		"esac": ColorKeyword, "echo": ColorKeyword, "read": ColorKeyword, "export": ColorKeyword,
	}

	// 7. Data / Markup (HTML, XML, JSON, SQL)
	markupData := map[string]tcell.Color{
		"true": ColorType, "false": ColorType, "null": ColorType,
		"SELECT": ColorKeyword, "FROM": ColorKeyword, "WHERE": ColorKeyword, "INSERT": ColorKeyword,
		"UPDATE": ColorKeyword, "DELETE": ColorKeyword, "JOIN": ColorKeyword, "ON": ColorKeyword,
	}

	// Asignar los mapas a sus respectivas extensiones
	extensions := map[string]map[string]tcell.Color{
		".c": cFamily, ".h": cFamily, ".cpp": cFamily, ".hpp": cFamily, ".cs": cFamily, ".java": cFamily, ".php": cFamily,
		".js": jsFamily, ".ts": jsFamily, ".jsx": jsFamily, ".tsx": jsFamily, ".mjs": jsFamily,
		".py": python, ".pyw": python, ".gd": python,
		".go": goLang,
		".rs": rust,
		".sh": shell, ".bash": shell, ".zsh": shell,
		".json": markupData, ".yml": markupData, ".yaml": markupData, ".sql": markupData,
	}

	for ext, dict := range extensions {
		keywords[ext] = dict
	}
}

// HighlightLine procesa una línea y devuelve un array de estilos.
func HighlightLine(line string, filename string)[]tcell.Style {
	ext := filepath.Ext(filename)
	kwMap, ok := keywords[ext]
	if !ok {
		kwMap = map[string]tcell.Color{}
	}

	runes :=[]rune(line)
	styles := make([]tcell.Style, len(runes))
	defaultStyle := tcell.StyleDefault.Foreground(ColorDefault)

	for i := 0; i < len(styles); i++ {
		styles[i] = defaultStyle
	}

	i := 0
	for i < len(runes) {
		// Comentarios (// o #)
		if (i < len(runes)-1 && runes[i] == '/' && runes[i+1] == '/') || runes[i] == '#' {
			for j := i; j < len(runes); j++ {
				styles[j] = tcell.StyleDefault.Foreground(ColorComment)
			}
			break
		}

		// Strings (" ", ' ', ` `)
		if runes[i] == '"' || runes[i] == '\'' || runes[i] == '`' {
			quote := runes[i]
			styles[i] = tcell.StyleDefault.Foreground(ColorString)
			i++
			for i < len(runes) && runes[i] != quote {
				styles[i] = tcell.StyleDefault.Foreground(ColorString)
				i++
			}
			if i < len(runes) {
				styles[i] = tcell.StyleDefault.Foreground(ColorString)
				i++
			}
			continue
		}

		// Números
		if unicode.IsDigit(runes[i]) {
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				styles[i] = tcell.StyleDefault.Foreground(ColorNumber)
				i++
			}
			continue
		}

		// Palabras y Tipos Clave
		if unicode.IsLetter(runes[i]) || runes[i] == '_' {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])
			if color, found := kwMap[word]; found {
				for j := start; j < i; j++ {
					styles[j] = tcell.StyleDefault.Foreground(color)
				}
			}
			continue
		}
		i++
	}

	return styles
}