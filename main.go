package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-graphviz"
)

const graphImage = "./graph.svg" // Файл, в который сохраняем изображение графа.

// Program ...
type Program struct {
	Name   string     `json:"program"`
	Input  [][]string `json:"input"`
	Output []string   `json:"output"`
	Childs []Program  // Массив дочерних программ.
}

func main() {
	// Читаем программы из шаблонов.
	programs, err := readTemplates("./templates/")
	if err != nil {
		log.Fatalln(err)
	}
	// Сохраним изображение графа в файл ./graph.svg.
	if err := drawGraph(programs, graphImage); err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Граф находится в файле:", graphImage)
}

// readTemplates читает все JSON файлы из директории и возвращает массив программ []Program.
func readTemplates(dir string) ([]Program, error) {
	var programs []Program
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// пропускаем файлы, которые не json.
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		jsonData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var program Program
		if err := json.Unmarshal(jsonData, &program); err != nil {
			return err
		}
		programs = append(programs, program)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Проставим связи между программами.
	for motherIdx, mother := range programs {
		for _, child := range programs {
			if mother.Name != child.Name && isChild(mother, child) {
				programs[motherIdx].Childs = append(programs[motherIdx].Childs, child)
			}
		}
	}
	return programs, err
}

// isChild вернет true, если выходные параметры родителя входят в один из требуемых наборов входных параметров ребенка, иначе false.
func isChild(mother Program, child Program) bool {
	for _, childInputs := range child.Input {
		isChild := true
		for _, in := range childInputs {
			if !slices.Contains(mother.Output, in) {
				isChild = false
				break
			}
		}
		if isChild {
			return true
		}
	}
	return false
}

// drawGraph нарисует граф в SVG-формате и запишет его в файл.
func drawGraph(programs []Program, filename string) error {
	g := graphviz.New()
	graph, err := g.Graph()
	if err != nil {
		return nil
	}
	defer graph.Close()
	for _, p := range programs {
		n, err := graph.CreateNode(p.Name)
		if err != nil {
			return nil
		}
		for _, c := range p.Childs {
			m, err := graph.CreateNode(c.Name)
			if err != nil {
				return nil
			}
			e, err := graph.CreateEdge(fmt.Sprintf("%s_%s", p.Name, c.Name), n, m)
			if err != nil {
				return nil
			}
			e.SetLabel(fmt.Sprintf("[%s]", strings.Join(p.Output, ", ")))
		}
	}
	return g.RenderFilename(graph, graphviz.SVG, filename)
}
