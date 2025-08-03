package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
)

//go:embed prelude.tape
var prelude string

var flags struct {
	SampleRate int
	BPM        float64
	TPB        int
	EvalFile   string
	EvalScript string
}

func SampleRate() int {
	return flags.SampleRate
}

type App struct {
	vm            *VM
	openFiles     map[string]string
	currentFile   string
	shouldExit    bool
	font          *Font
	tm            *TileMap
	ts            *TileScreen
	editor        *Editor
	lastScript    []byte
	result        Val
	tapeDisplay   *TapeDisplay
	tapePlayer    *OtoPlayer
	globalKeyMap  *KeyMap
	editorKeyMap  *KeyMap
	currentKeyMap *KeyMap
}

func (app *App) Init() error {
	slog.Debug("Init")
	err := InitOtoContext(SampleRate())
	if err != nil {
		return err
	}
	font, err := LoadFontFromFile("/usr/share/fonts/droid/DroidSansMono.ttf")
	if err != nil {
		return err
	}
	app.font = font
	face, err := font.GetFace(12)
	if err != nil {
		return err
	}
	tileMapSize := Size{X: 16, Y: 32}
	faceImage, err := font.GetFaceImage(face, tileMapSize.X, tileMapSize.Y)
	if err != nil {
		return err
	}
	tm, err := CreateTileMap(faceImage, tileMapSize.X, tileMapSize.Y)
	if err != nil {
		return err
	}
	app.tm = tm
	ts, err := tm.CreateScreen()
	if err != nil {
		return err
	}
	app.ts = ts
	tapeScript := ""
	if app.currentFile != "" {
		tapeScript = app.openFiles[app.currentFile]
	}
	app.editor = CreateEditor(tapeScript)
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return err
	}
	app.tapeDisplay = tapeDisplay
	executeEditorScriptIfChanged := func() {
		editorScript := app.editor.GetBytes()
		if slices.Compare(editorScript, app.lastScript) == 0 {
			return
		}
		vm := app.vm
		vm.Reset()
		vm.DoPushEnv()
		tapePath := "<temp-tape>"
		if app.currentFile != "" {
			tapePath = app.currentFile
		}
		app.lastScript = app.editor.GetBytes()
		err := vm.ParseAndExecute(bytes.NewReader(app.lastScript), tapePath)
		if err != nil {
			slog.Error("parse error", "err", err)
			app.result = err
		} else {
			app.result = vm.Pop()
		}
	}
	globalKeyMap := CreateKeyMap()
	globalKeyMap.Bind("C-p", CreateKeyHandler(func() {
		executeEditorScriptIfChanged()
		if tape, ok := app.result.(*Tape); ok {
			if app.tapePlayer != nil {
				app.tapePlayer.Close()
			}
			player := otoContext.NewPlayer(MakeTapeReader(tape, 2))
			player.Play()
			app.tapePlayer = player
		}
	}))
	app.globalKeyMap = &globalKeyMap
	editorKeyMap := CreateKeyMap()
	editorKeyMap.Bind("Escape", CreateKeyHandler(app.Reset))
	editorKeyMap.Bind("C-g", CreateKeyHandler(app.Reset))
	editorKeyMap.Bind("Enter", CreateKeyHandler(func() {
		DispatchAction(func() UndoFunc {
			app.editor.SplitLine()
			return func() {
				app.editor.AdvanceColumn(-1)
				app.editor.DeleteRune()
			}
		})
	}))
	editorKeyMap.Bind("Left", CreateKeyHandler(func() {
		app.editor.AdvanceColumn(-1)
	}))
	editorKeyMap.Bind("Right", CreateKeyHandler(func() {
		app.editor.AdvanceColumn(1)
	}))
	editorKeyMap.Bind("Up", CreateKeyHandler(func() {
		app.editor.AdvanceLine(-1)
	}))
	editorKeyMap.Bind("Down", CreateKeyHandler(func() {
		app.editor.AdvanceLine(1)
	}))
	editorKeyMap.Bind("PageUp", CreateKeyHandler(func() {
		for range app.editor.height {
			app.editor.AdvanceLine(-1)
		}
	}))
	editorKeyMap.Bind("PageDown", CreateKeyHandler(func() {
		for range app.editor.height {
			app.editor.AdvanceLine(1)
		}
	}))
	editorKeyMap.Bind("Delete", CreateKeyHandler(func() {
		DispatchAction(func() UndoFunc {
			deletedRune := app.editor.DeleteRune()
			return func() {
				if deletedRune != 0 {
					app.editor.InsertRune(deletedRune)
					app.editor.AdvanceColumn(-1)
				}
			}
		})
	}))
	editorKeyMap.Bind("Backspace", CreateKeyHandler(func() {
		if app.editor.AtBOF() {
			return
		}
		DispatchAction(func() UndoFunc {
			app.editor.AdvanceColumn(-1)
			deletedRune := app.editor.DeleteRune()
			return func() {
				if deletedRune != 0 {
					app.editor.InsertRune(deletedRune)
				}
			}
		})
	}))
	editorKeyMap.Bind("Home", CreateKeyHandler(app.editor.MoveToBOL))
	editorKeyMap.Bind("End", CreateKeyHandler(app.editor.MoveToEOL))
	editorKeyMap.Bind("Tab", CreateKeyHandler(app.editor.InsertSpacesUntilNextTabStop))
	editorKeyMap.Bind("C-Enter", CreateKeyHandler(func() {
		executeEditorScriptIfChanged()
	}))
	editorKeyMap.Bind("C-z", CreateKeyHandler(UndoLastAction))
	editorKeyMap.Bind("C-q", CreateKeyHandler(app.Quit))
	editorKeyMap.Bind("C-Left", CreateKeyHandler(app.editor.WordLeft))
	editorKeyMap.Bind("C-Right", CreateKeyHandler(app.editor.WordRight))
	editorKeyMap.Bind("C-a", CreateKeyHandler(app.editor.MoveToBOL))
	editorKeyMap.Bind("C-e", CreateKeyHandler(app.editor.MoveToEOL))
	editorKeyMap.Bind("C-Home", CreateKeyHandler(app.editor.MoveToBOF))
	editorKeyMap.Bind("C-End", CreateKeyHandler(app.editor.MoveToEOF))
	editorKeyMap.Bind("C-k", CreateKeyHandler(func() {
		if app.editor.AtEOL() {
			app.editor.DeleteRune()
		} else {
			for !app.editor.AtEOL() {
				app.editor.DeleteRune()
			}
		}
	}))
	editorKeyMap.Bind("C-Backspace", CreateKeyHandler(func() {
		DispatchAction(func() UndoFunc {
			app.editor.SetMark()
			app.editor.WordLeft()
			deletedRunes := app.editor.KillRegion()
			return func() {
				app.editor.InsertRunes(deletedRunes)
			}
		})
	}))
	editorKeyMap.Bind("C-u", CreateKeyHandler(func() {
		DispatchAction(func() UndoFunc {
			app.editor.SetMark()
			app.editor.MoveToBOL()
			deletedRunes := app.editor.KillRegion()
			return func() {
				app.editor.InsertRunes(deletedRunes)
			}
		})
	}))
	editorKeyMap.Bind("C-Space", CreateKeyHandler(app.editor.SetMark))
	editorKeyMap.Bind("C-w", CreateKeyHandler(func() {
		DispatchAction(func() UndoFunc {
			p, _ := app.editor.PointAndMarkInOrder()
			deletedRunes := app.editor.KillRegion()
			return func() {
				app.editor.SetPoint(p)
				app.editor.InsertRunes(deletedRunes)
			}
		})
	}))
	editorKeyMap.Bind("C-y", CreateKeyHandler(func() {
		DispatchAction(func() UndoFunc {
			p0 := app.editor.GetPoint()
			app.editor.Paste()
			p1 := app.editor.GetPoint()
			return func() {
				app.editor.KillBetween(p0, p1)
			}
		})
	}))
	editorKeyMap.Bind("C-s", CreateKeyHandler(func() {
		if app.currentFile != "" {
			os.WriteFile(app.currentFile, app.editor.GetBytes(), 0o644)
		}
	}))
	editorKeyMap.Bind("M-b", CreateKeyHandler(app.editor.WordLeft))
	editorKeyMap.Bind("M-f", CreateKeyHandler(app.editor.WordRight))
	editorKeyMap.Bind("M-w", CreateKeyHandler(app.editor.YankRegion))
	editorKeyMap.Bind("M-Backspace", CreateKeyHandler(func() {
		app.editor.SetMark()
		app.editor.WordLeft()
		app.editor.KillRegion()
	}))
	app.editorKeyMap = &editorKeyMap
	app.currentKeyMap = app.editorKeyMap
	return nil
}

func (app *App) IsRunning() bool {
	return !app.shouldExit
}

func (app *App) Quit() {
	app.shouldExit = true
}

func (app *App) OnKey(key glfw.Key, scancode int, action glfw.Action, modes glfw.ModifierKey) {
	//slog.Debug("OnKey", "key", key, "scancode", scancode, "action", action, "modes", modes)
	if action != glfw.Press && action != glfw.Repeat {
		return
	}
	var keyName string
	switch key {
	case glfw.KeySpace:
		keyName = "Space"
	case glfw.KeyEscape:
		keyName = "Escape"
	case glfw.KeyEnter:
		keyName = "Enter"
	case glfw.KeyTab:
		keyName = "Tab"
	case glfw.KeyBackspace:
		keyName = "Backspace"
	case glfw.KeyInsert:
		keyName = "Insert"
	case glfw.KeyDelete:
		keyName = "Delete"
	case glfw.KeyRight:
		keyName = "Right"
	case glfw.KeyLeft:
		keyName = "Left"
	case glfw.KeyDown:
		keyName = "Down"
	case glfw.KeyUp:
		keyName = "Up"
	case glfw.KeyPageUp:
		keyName = "PageUp"
	case glfw.KeyPageDown:
		keyName = "PageDown"
	case glfw.KeyHome:
		keyName = "Home"
	case glfw.KeyEnd:
		keyName = "End"
	case glfw.KeyF1:
		keyName = "F1"
	case glfw.KeyF2:
		keyName = "F2"
	case glfw.KeyF3:
		keyName = "F3"
	case glfw.KeyF4:
		keyName = "F4"
	case glfw.KeyF5:
		keyName = "F5"
	case glfw.KeyF6:
		keyName = "F6"
	case glfw.KeyF7:
		keyName = "F7"
	case glfw.KeyF8:
		keyName = "F8"
	case glfw.KeyF9:
		keyName = "F9"
	case glfw.KeyF10:
		keyName = "F10"
	case glfw.KeyF11:
		keyName = "F11"
	case glfw.KeyF12:
		keyName = "F12"
	default:
		keyName = glfw.GetKeyName(key, scancode)
	}
	if modes&glfw.ModShift != 0 {
		keyName = "S-" + keyName
	}
	if modes&glfw.ModAlt != 0 {
		keyName = "M-" + keyName
	}
	if modes&glfw.ModControl != 0 {
		keyName = "C-" + keyName
	}
	if app.globalKeyMap != nil && app.globalKeyMap.HandleKey(keyName) {
		return
	}
	if app.currentKeyMap != nil && app.currentKeyMap.HandleKey(keyName) {
		return
	}
}

func (app *App) OnChar(char rune) {
	//slog.Debug("OnChar", "char", char)
	app.editor.InsertRune(char)
}

func (app *App) OnFramebufferSize(width, height int) {
	slog.Debug("OnFramebufferSize", "width", width, "height", height)
}

func (app *App) Render() error {
	ts := app.ts
	ts.Clear()
	screenPane := ts.GetPane()
	switch result := app.result.(type) {
	case error:
		editorPane, statusPane := screenPane.SplitY(-1)
		app.editor.Render(editorPane)
		statusPane.WithFgBg(ColorWhite, ColorRed, func() {
			statusPane.DrawString(0, 0, result.Error())
		})
	case Str:
		editorPane, statusPane := screenPane.SplitY(-1)
		app.editor.Render(editorPane)
		statusPane.DrawString(0, 0, string(result))
	case Num:
		editorPane, statusPane := screenPane.SplitY(-1)
		app.editor.Render(editorPane)
		statusPane.DrawString(0, 0, strconv.FormatFloat(float64(result), 'f', -1, 64))
	case *Tape:
		editorPane, tapeDisplayPane := screenPane.SplitY(-8)
		app.editor.Render(editorPane)
		app.tapeDisplay.Render(result, tapeDisplayPane.GetPixelRect(), result.nframes, 0)
	default:
		editorPane, statusPane := screenPane.SplitY(-1)
		app.editor.Render(editorPane)
		statusPane.DrawString(0, 0, fmt.Sprintf("%#v", result))
	}
	ts.Render()
	return nil
}

func (app *App) Update() error {
	return nil
}

func (app *App) Reset() {
	if app.tapePlayer != nil {
		app.tapePlayer.Close()
		app.tapePlayer = nil
	}
	app.editor.ResetState()
}

func (app *App) Close() error {
	slog.Debug("Close")
	app.Reset()
	app.ts.Close()
	app.tm.Close()
	app.editor.Close()
	return nil
}

func runGui(vm *VM, openFiles map[string]string, currentFile string) error {
	app := &App{
		vm:          vm,
		openFiles:   openFiles,
		currentFile: currentFile,
	}
	var windowTitle string
	if currentFile != "" {
		windowTitle = fmt.Sprintf("mixtape : %s", currentFile)
	} else {
		windowTitle = "mixtape"
	}
	return WithGL(windowTitle, app)
}

func setDefaults(vm *VM) {
	vm.SetVal(":bpm", flags.BPM)
	vm.SetVal(":tpb", flags.TPB)

	beatsPerSecond := flags.BPM / 60.0
	framesPerBeat := float64(SampleRate()) / beatsPerSecond
	vm.SetVal(":nf", int(framesPerBeat))

	vm.SetVal(":freq", 440)
	vm.SetVal(":phase", 0)
	vm.SetVal(":width", 0.5)
}

func runWithArgs(vm *VM, args []string) error {
	openFiles := make(map[string]string)
	currentFile := ""
	if flags.EvalScript != "" {
		return vm.ParseAndExecute(strings.NewReader(flags.EvalScript), "<script>")
	}
	if flags.EvalFile != "" {
		data, err := os.ReadFile(flags.EvalFile)
		if err != nil {
			return err
		}
		return vm.ParseAndExecute(bytes.NewReader(data), flags.EvalFile)
	}
	for _, arg := range args {
		data, err := os.ReadFile(arg)
		if err != nil {
			return err
		}
		openFiles[arg] = string(data)
		currentFile = arg
	}
	return runGui(vm, openFiles, currentFile)
}

func main() {
	var vm *VM
	var err error
	flag.IntVar(&flags.SampleRate, "sr", 48000, "Sample rate")
	flag.Float64Var(&flags.BPM, "bpm", 120, "Beats per minute")
	flag.IntVar(&flags.TPB, "tpb", 96, "Ticks per beat")
	flag.StringVar(&flags.EvalFile, "f", "", "File to evaluate")
	flag.StringVar(&flags.EvalScript, "e", "", "Script to evaluate")
	flag.Parse()
	vm, err = CreateVM()
	if err != nil {
		slog.Error("vm initialization error", "err", err)
		os.Exit(1)
	}
	setDefaults(vm)
	err = vm.ParseAndExecute(strings.NewReader(prelude), "<prelude>")
	if err != nil {
		slog.Error("error while parsing the prelude", "err", err)
	}
	err = runWithArgs(vm, flag.Args())
	if err != nil {
		slog.Error("vm error", "err", err)
	}
}
