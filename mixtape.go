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
	editor      *Editor
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
	mixScript, err := os.ReadFile(app.mixFilePath)
	if err != nil {
		return err
	}
	app.editor = CreateEditor(string(mixScript))
	return nil
}

func (app *App) IsRunning() bool {
	return app.isRunning
}

func (app *App) OnKey(key glfw.Key, scancode int, action glfw.Action, modes glfw.ModifierKey) {
	//slog.Info("OnKey", "key", key, "scancode", scancode, "action", action, "modes", modes)
	if action == glfw.Press || action == glfw.Repeat {
		if modes == 0 {
			switch key {
			case glfw.KeyEscape:
				app.editor.Quit()
			case glfw.KeyEnter:
				app.editor.SplitLine()
			case glfw.KeyLeft:
				app.editor.AdvanceColumn(-1)
			case glfw.KeyRight:
				app.editor.AdvanceColumn(1)
			case glfw.KeyUp:
				app.editor.AdvanceLine(-1)
			case glfw.KeyDown:
				app.editor.AdvanceLine(1)
			case glfw.KeyPageUp:
				for range app.editor.height {
					app.editor.AdvanceLine(-1)
				}
			case glfw.KeyPageDown:
				for range app.editor.height {
					app.editor.AdvanceLine(1)
				}
			case glfw.KeyDelete:
				app.editor.DeleteRune()
			case glfw.KeyBackspace:
				if !app.editor.AtBOF() {
					app.editor.AdvanceColumn(-1)
					app.editor.DeleteRune()
				}
			case glfw.KeyHome:
				app.editor.MoveToBOL()
			case glfw.KeyEnd:
				app.editor.MoveToEOL()
			}
		}
		if modes&glfw.ModControl == glfw.ModControl {
			switch key {
			case glfw.KeyQ:
				app.isRunning = false
			case glfw.KeyLeft:
				app.editor.WordLeft()
			case glfw.KeyRight:
				app.editor.WordRight()
			case glfw.KeyA:
				app.editor.MoveToBOL()
			case glfw.KeyE:
				app.editor.MoveToEOL()
			case glfw.KeyHome:
				app.editor.MoveToBOF()
			case glfw.KeyEnd:
				app.editor.MoveToEOF()
			case glfw.KeyK:
				if app.editor.AtEOL() {
					app.editor.DeleteRune()
				} else {
					for !app.editor.AtEOL() {
						app.editor.DeleteRune()
					}
				}
			case glfw.KeyBackspace:
				app.editor.SetMark()
				app.editor.WordLeft()
				app.editor.KillRegion()
			case glfw.KeyU:
				for !app.editor.AtBOL() {
					app.editor.AdvanceColumn(-1)
					app.editor.DeleteRune()
				}
			case glfw.KeySpace:
				app.editor.SetMark()
			case glfw.KeyW:
				app.editor.KillRegion()
			case glfw.KeyY:
				app.editor.Paste()
			case glfw.KeyG:
				app.editor.Quit()
			case glfw.KeyS:
				os.WriteFile(app.mixFilePath, app.editor.GetBytes(), 0o644)
			}
		}
		if modes&glfw.ModAlt == glfw.ModAlt {
			switch key {
			case glfw.KeyW:
				app.editor.YankRegion()
			}
		}
	}
}

func (app *App) OnChar(char rune) {
	//slog.Info("OnChar", "char", char)
	app.editor.InsertRune(char)
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
	app.editor.Render(bottom)
	ts.Render()
	return nil
}

func (app *App) Update() error {
	return nil
}

func (app *App) Close() error {
	slog.Info("Close")
	app.ts.Close()
	app.tm.Close()
	app.editor.Close()
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
