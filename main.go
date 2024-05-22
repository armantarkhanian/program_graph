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
	"github.com/goccy/go-graphviz/cgraph"
)

const graphImage = "./graph.svg" // Файл, в который сохраняем изображение графа.

var userInput []string // Парметры, которые ввел пользователь (domain, IP, url и т.д.)

// Program ...
type Program struct {
	Name     string            `json:"program"`
	Input    [][]string        `json:"input"`
	Output   []string          `json:"output"`
	Commands []string          `json:"commands"`
	Comments []string          `json:"comments"`
	Filter   string            `json:"filter"`
	Regex    map[string]string `json:"regex"`

	childs []Program // Массив дочерних программ.
}

func main() {
	// Читаем программы из шаблонов.
	programs, err := readTemplates("./templates/")
	if err != nil {
		log.Fatalln(err)
	}
	if len(os.Args) >= 2 {
		// Если пользователь указал входные параметры, то построим граф только для тех программ, которые принимают на вход эти параметры.
		userInput = os.Args[1:]
		var userPrograms []Program
		for _, p := range programs {
			if isChild(Program{Output: userInput}, p) {
				userPrograms = append(userPrograms, p)
			}
		}
		if len(userPrograms) == 0 {
			log.Fatalf("Нет программ, которые бы принимали на вход: [%s]\n", strings.Join(userInput, ", "))
		}
		programs = userPrograms
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
	connectProgramms(programs)
	connectProgramms(programs)
	return programs, err
}

func connectProgramms(programs []Program) {
	for motherIdx, mother := range programs {
		for childIdx, child := range programs {
			if mother.Name != child.Name && isChild(mother, child) {
				programs[childIdx].Output = appendOutput(programs[childIdx].Output, mother.Output)
				programs[motherIdx].childs = append(programs[motherIdx].childs, child)
			}
		}
	}
}

// TODO: возможно не понадобится.
func appendOutput(oldOutput, newOutput []string) []string {
	uniqueParams := make(map[string]struct{})
	for _, p := range oldOutput {
		uniqueParams[p] = struct{}{}
	}
	for _, p := range newOutput {
		uniqueParams[p] = struct{}{}
	}
	var output []string
	for p := range uniqueParams {
		output = append(output, p)
	}
	return output
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
	root, err := graph.CreateNode("user_input")
	if err != nil {
		return err
	}
	root.SetLabel(fmt.Sprintf("На вход:\\n[%s]", strings.Join(userInput, ", ")))
	root.SetShape(cgraph.RectangleShape)
	for _, p := range programs {
		n, err := graph.CreateNode(p.Name)
		if err != nil {
			return nil
		}
		_, err = graph.CreateEdge("", root, n)
		if err != nil {
			return nil
		}
		for _, c := range p.childs {
			m, err := graph.CreateNode(c.Name)
			if err != nil {
				return nil
			}
			edgeNode, err := graph.CreateNode(fmt.Sprintf("%s_%s", p.Name, c.Name))
			if err != nil {
				return nil
			}
			edgeNode.SetLabel(strings.Join(p.Output, ", "))
			edgeNode.SetShape(cgraph.RectangleShape)
			_, err = graph.CreateEdge("", n, edgeNode)
			if err != nil {
				return nil
			}
			_, err = graph.CreateEdge("", edgeNode, m)
			if err != nil {
				return nil
			}
		}
	}
	return g.RenderFilename(graph, graphviz.SVG, filename)
}
