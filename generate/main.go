package generate

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fujitsueos/go-server-generator/templates"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	log "github.com/sirupsen/logrus"
)

func init() {
	if err := exec.Command("goimports").Run(); err != nil {
		log.Fatal("Could not run goimports, make sure it's installed.\nInstall it by running:\ngo get golang.org/x/tools/cmd/goimports")
	}
}

// FromSwagger generates a "generated" folder with generated code in the same folder as the swagger file
func FromSwagger(swaggerPath string) (err error) {
	var swagger *spec.Swagger

	if swagger, err = readValidSwagger(swaggerPath); err != nil {
		return
	}

	if err = generateServer(swaggerPath, swagger); err != nil {
		return
	}

	log.Info("Generation completed")

	return
}

func readValidSwagger(swaggerPath string) (swagger *spec.Swagger, err error) {
	var specDoc *loads.Document

	if specDoc, err = loads.Spec(swaggerPath); err != nil {
		return
	}

	if err = validate.Spec(specDoc, strfmt.Default); err != nil {
		return
	}

	swagger = specDoc.Spec()

	return
}

func generateServer(path string, swagger *spec.Swagger) (err error) {
	paths := []string{
		"generated/swagger.go",
		"generated/model/model.go",
		"generated/model/validate.go",
		"generated/model/errors.go",
		"generated/model/routeerrors.go",
		"generated/router/router.go",
	}

	files := map[string]*os.File{}
	packages := map[string]string{}

	for _, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		var closeFile func()
		if files[name], packages[name], closeFile, err = createOutputFile(path, p); err != nil {
			return
		}
		defer closeFile()
	}

	// turn swagger file into a Go string
	if err = inlineSwaggerFile(path, files["swagger"]); err != nil {
		return
	}

	// create the model and write to the model and validate files
	var readOnlyTypes map[string]bool
	if readOnlyTypes, err = Model(files["model"], files["validate"], files["errors"], swagger.Definitions); err != nil {
		return
	}

	// create the router and write to the router file
	err = Router(files["router"], files["routeerrors"], swagger.Paths, readOnlyTypes, packages["model"])

	return
}

func createOutputFile(swaggerPath, relativePath string) (file *os.File, packageName string, closeFile func(), err error) {
	path := filepath.Join(filepath.Dir(swaggerPath), relativePath)
	folder := filepath.Dir(path)

	// create the folder and file
	if err = os.MkdirAll(folder, os.ModePerm); err != nil {
		return
	}
	if file, err = os.Create(path); err != nil {
		return
	}

	// get the name of the go package
	goSrcPath := filepath.Join(os.Getenv("GOPATH"), "src")
	if packageName, err = filepath.Rel(goSrcPath, folder); err != nil {
		return
	}

	closeFile = func() {
		log.Infof("Format file %s", path)
		if err := exec.Command("goimports", "-w", path).Run(); err != nil {
			log.Errorf("Could not format file %s", path)
		}
		if err := file.Close(); err != nil {
			log.Error(err)
		}
	}

	return
}

func inlineSwaggerFile(swaggerPath string, file io.Writer) (err error) {
	var swaggerFile io.Reader
	if swaggerFile, err = os.Open(swaggerPath); err != nil {
		return
	}

	var swaggerData []byte
	if swaggerData, err = ioutil.ReadAll(swaggerFile); err != nil {
		return
	}

	err = templates.Swagger.Execute(file, string(swaggerData))

	return
}
