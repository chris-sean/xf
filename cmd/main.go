package main

import (
	"github.com/chris-sean/xf/codegen"
	"os"
	"os/exec"
)

func main() {
	cmd := os.Args[1]
	switch cmd {
	case "scaffold":
		codegen.MustGenerateScaffold()
		exec.Command("go mod tidy").Run()
	}
}
