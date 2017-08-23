package generate

import (
	"os"
	"os/exec"
	"path/filepath"

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
	var (
		modelFile, validateFile, routerFile                *os.File
		modelPackage                                       string
		closeModelFile, closeValidateFile, closeRouterFile func()
	)

	// create the model file
	if modelFile, modelPackage, closeModelFile, err = createOutputFile(path, "generated/model/model.go"); err != nil {
		return
	}
	defer closeModelFile()

	// create the validate file
	if validateFile, _, closeValidateFile, err = createOutputFile(path, "generated/model/validate.go"); err != nil {
		return
	}
	defer closeValidateFile()

	// create the router file
	if routerFile, _, closeRouterFile, err = createOutputFile(path, "generated/router/router.go"); err != nil {
		return
	}
	defer closeRouterFile()

	// create the model and write to the model and validate files
	var readOnlyTypes map[string]bool
	if readOnlyTypes, err = Model(modelFile, validateFile, swagger.Definitions); err != nil {
		return
	}

	// create the router and write to the router file
	err = Router(routerFile, swagger.Paths, readOnlyTypes, modelPackage)

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
