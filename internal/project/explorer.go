package project

import (
	"os"
	"path/filepath"
)

type Node struct {
	Name  string
	Path  string
	IsDir bool
}

type Explorer struct {
	Cwd      string
	Nodes[]Node
	Selected int
}

func NewExplorer(path string) *Explorer {
	e := &Explorer{Cwd: path}
	e.Refresh()
	return e
}

func (e *Explorer) Refresh() {
	e.Nodes =[]Node{}
	
	if e.Cwd != "/" && e.Cwd != "." {
		e.Nodes = append(e.Nodes, Node{Name: "..", Path: filepath.Dir(e.Cwd), IsDir: true})
	}

	entries, err := os.ReadDir(e.Cwd)
	if err != nil {
		return
	}

	var dirs, files[]Node
	for _, entry := range entries {
		n := Node{
			Name:  entry.Name(),
			Path:  filepath.Join(e.Cwd, entry.Name()),
			IsDir: entry.IsDir(),
		}
		if n.IsDir {
			dirs = append(dirs, n)
		} else {
			files = append(files, n)
		}
	}

	e.Nodes = append(e.Nodes, dirs...)
	e.Nodes = append(e.Nodes, files...)

	if e.Selected >= len(e.Nodes) {
		e.Selected = 0
	}
}

func (e *Explorer) MoveUp() {
	if e.Selected > 0 {
		e.Selected--
	}
}

func (e *Explorer) MoveDown() {
	if e.Selected < len(e.Nodes)-1 {
		e.Selected++
	}
}

func (e *Explorer) CurrentNode() *Node {
	if len(e.Nodes) == 0 {
		return nil
	}
	return &e.Nodes[e.Selected]
}