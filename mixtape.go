package main

import (
	"bytes"
	"log"
	"os"
	"strings"
)

func runGui(vm *VM, mixFilePath string) error {
	return nil
}

func main() {
	vm := NewVM()
	var err error
	if len(os.Args) == 1 {
		err = vm.ParseAndExecute(os.Stdin, "<stdin>")
	} else {
		evalScript := false
		evalFile := false
		for _, arg := range os.Args[1:] {
			if evalScript {
				err = vm.ParseAndExecute(strings.NewReader(arg), "<script>")
				if err != nil {
					break
				}
				evalScript = false
				continue
			}
			if evalFile {
				data, err := os.ReadFile(arg)
				if err != nil {
					break
				}
				err = vm.ParseAndExecute(bytes.NewReader(data), arg)
				if err != nil {
					break
				}
				evalFile = false
				continue
			}
			switch arg {
			case "-e":
				evalScript = true
			case "-f":
				evalFile = true
			default:
				err = runGui(vm, arg)
				break
			}
		}
	}
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
