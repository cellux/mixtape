package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"strings"
)

type App struct {
	vm          *VM
	tapePath    string
	isRunning   bool
	font        *Font
	tm          *TileMap
	ts          *TileScreen
	editor      *Editor
	tape        *Tape
	tapeDisplay *TapeDisplay
}

func runGui(vm *VM, tapePath string) error {
	app := &App{
		vm:        vm,
		tapePath:  tapePath,
		isRunning: true,
	}
	var windowTitle string
	if tapePath != "" {
		windowTitle = fmt.Sprintf("mixtape : %s", tapePath)
	} else {
		windowTitle = "mixtape"
	}
	return WithGL(windowTitle, app)
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
	var tapeScript []byte
	if app.tapePath != "" {
		tapeScript, err = os.ReadFile(app.tapePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				tapeScript = []byte{}
			} else {
				return err
			}
		}
	} else {
		tapeScript = []byte{}
	}
	app.editor = CreateEditor(string(tapeScript))
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return err
	}
	app.tapeDisplay = tapeDisplay
	return nil
}

func (app *App) IsRunning() bool {
	return app.isRunning
}

func (app *App) Quit() {
	app.isRunning = false
}

func (app *App) OnKey(key glfw.Key, scancode int, action glfw.Action, modes glfw.ModifierKey) {
	//slog.Info("OnKey", "key", key, "scancode", scancode, "action", action, "modes", modes)
	if action == glfw.Press || action == glfw.Repeat {
		if modes == 0 {
			switch key {
			case glfw.KeyEscape:
				app.editor.ResetState()
			case glfw.KeyEnter:
				DispatchAction(
					func() UndoFunc {
						app.editor.SplitLine()
						return func() {
							app.editor.AdvanceColumn(-1)
							app.editor.DeleteRune()
						}
					})
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
				DispatchAction(
					func() UndoFunc {
						deletedRune := app.editor.DeleteRune()
						return func() {
							if deletedRune == 0 {
								return
							}
							if deletedRune == '\n' {
								app.editor.SplitLine()
							} else {
								app.editor.InsertRune(deletedRune)
							}
							app.editor.AdvanceColumn(-1)
						}
					})
			case glfw.KeyBackspace:
				if !app.editor.AtBOF() {
					DispatchAction(
						func() UndoFunc {
							app.editor.AdvanceColumn(-1)
							deletedRune := app.editor.DeleteRune()
							return func() {
								if deletedRune == 0 {
									return
								}
								if deletedRune == '\n' {
									app.editor.SplitLine()
								} else {
									app.editor.InsertRune(deletedRune)
								}
							}
						})
				}
			case glfw.KeyHome:
				app.editor.MoveToBOL()
			case glfw.KeyEnd:
				app.editor.MoveToEOL()
			case glfw.KeyTab:
				app.editor.InsertSpacesUntilNextTabStop()
			}
		}
		if modes&glfw.ModControl == glfw.ModControl {
			switch key {
			case glfw.KeyEnter:
				vm := app.vm
				vm.Reset()
				vm.PushEnv()
				tapePath := "<temp-tape>"
				if app.tapePath != "" {
					tapePath = app.tapePath
				}
				err := vm.ParseAndExecute(bytes.NewReader(app.editor.GetBytes()), tapePath)
				if err != nil {
					slog.Error("parse error", "error", err)
				} else {
					val := vm.PopVal()
					if tape, ok := val.(*Tape); ok {
						app.tape = tape
					} else {
						slog.Error(fmt.Sprintf("expected a Tape at top of stack, got %T", val))
					}
				}
			case glfw.KeyZ:
				UndoLastAction()
			case glfw.KeyQ:
				app.Quit()
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
				app.editor.ResetState()
			case glfw.KeyS:
				if app.tapePath != "" {
					os.WriteFile(app.tapePath, app.editor.GetBytes(), 0o644)
				}
			}
		}
		if modes&glfw.ModAlt == glfw.ModAlt {
			switch key {
			case glfw.KeyB:
				app.editor.WordLeft()
			case glfw.KeyF:
				app.editor.WordRight()
			case glfw.KeyW:
				app.editor.YankRegion()
			case glfw.KeyBackspace:
				app.editor.SetMark()
				app.editor.WordLeft()
				app.editor.KillRegion()
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
	topPane, bottomPane := tp.SplitY(5)
	editorPane, tapeDisplayPane := bottomPane.SplitY(-8)
	topPane.DrawString(0, 0, "Hello, world")
	app.editor.Render(editorPane)
	ts.Render()
	if app.tape != nil {
		app.tapeDisplay.Render(app.tape, tapeDisplayPane.GetPixelRect(), app.tape.nframes, 0)
	}
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

func processArgs(vm *VM, args []string) error {
	evalScript := false
	evalFile := false
	for _, arg := range args {
		if evalScript {
			err := vm.ParseAndExecute(strings.NewReader(arg), "<script>")
			if err != nil {
				return err
			}
			evalScript = false
			continue
		}
		if evalFile {
			data, err := os.ReadFile(arg)
			if err != nil {
				return err
			}
			err = vm.ParseAndExecute(bytes.NewReader(data), arg)
			if err != nil {
				return err
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
			return runGui(vm, arg)
		}
	}
	return nil
}

func main() {
	vm := NewVM()
	var err error
	if len(os.Args) == 1 {
		err = runGui(vm, "")
	} else {
		err = processArgs(vm, os.Args[1:])
	}
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
