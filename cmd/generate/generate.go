package main

import (
	"os"
	"path/filepath"
	"text/template"

	log "github.com/sirupsen/logrus"
)

var tpl = template.Must(template.New("generator").Parse(`package main

import (
	"os"
	"path"

	"github.com/fujitsueos/go-server-generator/generate"
)

func main() {
	generate.FromSwagger(path.Join(os.Getenv("GOPATH"), "{{ .SwaggerPath }}"))
}
`))

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Run this as: go run cmd/generator/main.go <swagger-file>")
	}

	swaggerPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	file, closeFile, err := createOutputFile(swaggerPath, "cmd/generate/generate.go")
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile()

	if err := generateCode(swaggerPath, file); err != nil {
		log.Fatal(err)
	}

	log.Info("Generator code generated")
}

func createOutputFile(swaggerPath, relativePath string) (file *os.File, closeFile func(), err error) {
	path := filepath.Join(filepath.Dir(swaggerPath), relativePath)
	folder := filepath.Dir(path)

	// create the folder and file
	if err = os.MkdirAll(folder, os.ModePerm); err != nil {
		return
	}
	if file, err = os.Create(path); err != nil {
		return
	}

	closeFile = func() {
		if err := file.Close(); err != nil {
			log.Error(err)
		}
	}

	return
}

func generateCode(swaggerPath string, file *os.File) error {
	relSwaggerPath, err := filepath.Rel(os.Getenv("GOPATH"), swaggerPath)
	if err != nil {
		return err
	}

	return tpl.Execute(file, struct{ SwaggerPath string }{relSwaggerPath})
}
