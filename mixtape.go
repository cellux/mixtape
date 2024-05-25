package main

import (
	"bytes"
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"log"
	"log/slog"
	"os"
	"strings"
)

type App struct {
	vm          *VM
	mixFilePath string
	isRunning   bool
	font        *Font
	tm          *TileMap
	ts          *TileScreen
}

func runGui(vm *VM, mixFilePath string) error {
	app := &App{
		vm:          vm,
		mixFilePath: mixFilePath,
		isRunning:   true,
	}
	return WithGL(fmt.Sprintf("mixtape : %s", mixFilePath), app)
}

func (app *App) Init() error {
	slog.Info("Init")
	font, err := LoadFontFromFile("/usr/share/fonts/droid/DroidSansMono.ttf")
	if err != nil {
		return err
	}
	app.font = font
	face, err := font.GetFace(12)
	if err != nil {
		return err
	}
	faceImage, err := font.GetFaceImage(face, 16, 32)
	if err != nil {
		return err
	}
	tm, err := CreateTileMap(faceImage, 16, 32)
	if err != nil {
		return err
	}
	app.tm = tm
	ts, err := tm.CreateScreen()
	if err != nil {
		return err
	}
	app.ts = ts
	return nil
}

func (app *App) IsRunning() bool {
	return app.isRunning
}

func (app *App) OnKey(key glfw.Key, scancode int, action glfw.Action, modes glfw.ModifierKey) {
	slog.Info("OnKey", "key", key, "scancode", scancode, "action", action, "modes", modes)
	if action == glfw.Press && key == glfw.KeyEscape {
		app.isRunning = false
	}
}

func (app *App) OnChar(char rune) {
	slog.Info("OnChar", "char", char)
}

func (app *App) OnFramebufferSize(width, height int) {
	slog.Info("OnFramebufferSize", "width", width, "height", height)
}

func (app *App) Render() error {
	ts := app.ts
	ts.Clear()
	tp := ts.GetPane()
	top, bottom := tp.SplitY(5)
	top.DrawString(0, 0, "Hello, world")
	for x := range 512 {
		for y := range 512 {
			bottom.DrawRune(x, y, rune(x))
		}
	}
	if err := ts.Render(); err != nil {
		return err
	}
	return nil
}

func (app *App) Update() error {
	return nil
}

func (app *App) Close() error {
	slog.Info("Close")
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
