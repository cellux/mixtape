package main

import (
	"bytes"
	"embed"
	"errors"

	"github.com/go-gl/glfw/v3.3/glfw"
)

//go:embed assets/*
var assets embed.FS

// Event is the type of callback functions sent to the app's events channel
type Event func()

type App struct {
	vm            *VM
	openFiles     map[string]string
	currentFile   string
	shouldExit    bool
	font          *Font
	tm            *TileMap
	ts            *TileScreen
	screens       []Screen
	currentScreen int
	oto           *OtoState
	// rTape points to the currently rendered tape
	rTape        *Tape
	rTotalFrames int
	rDoneFrames  int
	kmm          *KeyMapManager
	events       chan Event
}

func (app *App) CurrentScreen() Screen {
	return app.screens[app.currentScreen]
}

func (app *App) postEvent(ev Event, dropIfFull bool) {
	if dropIfFull {
		select {
		case app.events <- ev:
		default:
		}
	} else {
		app.events <- ev
	}
}

func (app *App) Init() error {
	// Event queue used by background evaluation to post updates to the main thread.
	if app.events == nil {
		app.events = make(chan Event, 1024)
	}
	oto, err := NewOtoState(SampleRate())
	if err != nil {
		return err
	}
	app.oto = oto
	fontBytes, err := assets.ReadFile("assets/DroidSansMono.ttf")
	if err != nil {
		return err
	}
	font, err := LoadFontFromBytes(fontBytes)
	if err != nil {
		return err
	}
	app.font = font
	face, err := font.GetFace(14, contentScale)
	if err != nil {
		return err
	}
	sizeInTiles := Size{X: 16, Y: 32}
	faceImage, err := font.GetFaceImage(face, sizeInTiles)
	if err != nil {
		return err
	}
	tm, err := CreateTileMap(faceImage, sizeInTiles)
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

	helpBytes, err := assets.ReadFile("assets/help.txt")
	if err != nil {
		return err
	}

	globalKeyMap := CreateKeyMap(nil)
	globalKeyMap.Bind("C-g", app.Reset)
	globalKeyMap.Bind("Escape", app.Reset)
	globalKeyMap.Bind("C-z", UndoLastAction)
	globalKeyMap.Bind("C-x u", UndoLastAction)
	globalKeyMap.Bind("C-S--", UndoLastAction)
	globalKeyMap.Bind("C-q", app.Quit)

	helpScreen, err := CreateHelpScreen(app, globalKeyMap, string(helpBytes))
	if err != nil {
		return err
	}

	editScreen, err := CreateEditScreen(app, globalKeyMap, tapeScript)
	if err != nil {
		return err
	}
	app.screens = []Screen{helpScreen, editScreen}
	app.currentScreen = 1
	app.kmm = CreateKeyMapManager()
	app.kmm.SetCurrentKeyMap(app.CurrentScreen().Keymap())

	app.vm.tapeProgressCallback = func(t *Tape, nftotal, nfdone int) {
		app.postEvent(func() {
			if app.vm.IsEvaluating() {
				app.rTape = t
				app.rTotalFrames = nftotal
				app.rDoneFrames = nfdone
			}
		}, true)
	}
	return nil
}

func (app *App) IsRunning() bool {
	return !app.shouldExit
}

func (app *App) Quit() {
	app.shouldExit = true
}

func (app *App) SelectScreen(index int) {
	if index < 0 || index >= len(app.screens) {
		return
	}
	app.currentScreen = index
	app.kmm.SetCurrentKeyMap(app.CurrentScreen().Keymap())
}

func (app *App) OnKey(key glfw.Key, scancode int, action glfw.Action, modes glfw.ModifierKey) {
	//logger.Debug("OnKey", "key", key, "scancode", scancode, "action", action, "modes", modes)
	if action != glfw.Press && action != glfw.Repeat {
		return
	}
	// Screen switching via function keys (F1, F2, ...)
	if key >= glfw.KeyF1 && key <= glfw.KeyF12 {
		index := int(key - glfw.KeyF1)
		app.SelectScreen(index)
		return
	}
	var keyName string
	switch key {
	case glfw.KeyLeftShift, glfw.KeyLeftControl, glfw.KeyLeftAlt, glfw.KeyLeftSuper:
		return
	case glfw.KeyRightShift, glfw.KeyRightControl, glfw.KeyRightAlt, glfw.KeyRightSuper:
		return
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
	app.kmm.HandleKey(keyName)
}

func (app *App) OnChar(char rune) {
	//logger.Debug("OnChar", "char", char)
	if app.kmm.IsInsideKeySequence() {
		return
	}
	if cs, ok := app.CurrentScreen().(CharScreen); ok {
		cs.OnChar(app, char)
	}
}

func (app *App) OnFramebufferSize(width, height int) {
	logger.Debug("OnFramebufferSize", "width", width, "height", height)
}

func (app *App) Render() error {
	ts := app.ts
	ts.Clear()
	app.CurrentScreen().Render(app, ts)
	ts.Render()
	return nil
}

func (app *App) drainEvents() {
	for {
		select {
		case ev := <-app.events:
			ev()
		default:
			return // nothing queued right now
		}
	}
}

func (app *App) Update() error {
	app.drainEvents()
	return nil
}

func (app *App) evalEditorScript(editorScript []byte, evalSuccessCallback func()) {
	_, ok := app.CurrentScreen().(*EditScreen)
	if !ok {
		return
	}
	app.Reset()
	tapePath := "<temp-tape>"
	if app.currentFile != "" {
		tapePath = app.currentFile
	}
	go func() {
		vm := app.vm
		vm.DoPushEnv()
		var result Val
		err := vm.ParseAndEval(bytes.NewReader(editorScript), tapePath)
		if err != nil {
			if errors.Is(err, ErrEvalCancelled) {
				return
			}
			logger.Error("parse error", "err", err)
			result = makeErr(err)
		} else {
			result = vm.evalResult
			if streamable, ok := result.(Streamable); ok {
				stream := streamable.Stream()
				if stream.nframes > 0 {
					result = stream.Take(nil, stream.nframes)
				}
			}
		}
		app.postEvent(func() {
			app.rTape = nil
			app.rTotalFrames = 0
			app.rDoneFrames = 0
			if err != nil {
			} else {
				if evalSuccessCallback != nil {
					evalSuccessCallback()
				}
			}
		}, false)
	}()
}

func (app *App) playEvalResult() {
	app.oto.PlayTape(app.vm.evalResult)
}

func (app *App) Reset() {
	if app.vm.IsEvaluating() {
		app.vm.CancelEvaluation()
	}
	app.rTape = nil
	app.rTotalFrames = 0
	app.rDoneFrames = 0
	app.drainEvents()
	app.oto.StopAllPlayers()
	for _, screen := range app.screens {
		screen.Reset()
	}
	app.kmm.Reset()
}

func (app *App) Close() {
	logger.Debug("Close")
	app.Reset()
	app.ts.Close()
	app.tm.Close()
	for _, screen := range app.screens {
		screen.Close()
	}
}
