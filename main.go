package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fujitsueos/go-server-generator/generate"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Use this as: go-server-generator <swagger-file>")
	}

	swaggerPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if err := generate.FromSwagger(swaggerPath); err != nil {
		log.Fatal(err)
	}
}
