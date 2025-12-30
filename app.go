package main

import (
	"bytes"
	"embed"

	"github.com/go-gl/glfw/v3.3/glfw"
)

//go:embed assets/*
var assets embed.FS

// Event is the type of callback functions sent to the app's events channel
type Event func()

const (
	defaultFontSize FontSizeInPoints = 14
	minFontSize     FontSizeInPoints = 8
	maxFontSize     FontSizeInPoints = 72
	fontSizeStep    FontSizeInPoints = 1
)

type App struct {
	vm                 *VM
	openFiles          map[string]string
	currentFile        string
	shouldExit         bool
	font               *Font
	fontSize           FontSizeInPoints
	tm                 *TileMap
	ts                 *TileScreen
	screens            []Screen
	currentScreenIndex int
	oto                *OtoState
	// rTape points to the currently rendered tape
	rTape             *Tape
	rTotalFrames      int
	rDoneFrames       int
	globalKeyMap      KeyMap
	currentKeyHandler KeyHandler
	chordHandler      KeyHandler
	events            chan Event
}

func (app *App) reloadFont() error {
	face, err := app.font.GetFace(app.fontSize, contentScale)
	if err != nil {
		return err
	}
	sizeInTiles := Size{X: 16, Y: 32}
	faceImage, err := app.font.GetFaceImage(face, sizeInTiles)
	if err != nil {
		return err
	}
	tm, err := CreateTileMap(faceImage, sizeInTiles)
	if err != nil {
		return err
	}
	ts, err := tm.CreateScreen()
	if err != nil {
		tm.Close()
		return err
	}
	if app.ts != nil {
		app.ts.Close()
	}
	if app.tm != nil {
		app.tm.Close()
	}
	app.tm = tm
	app.ts = ts
	return nil
}

func (app *App) setFontSize(size FontSizeInPoints) {
	clamped := size
	if clamped < minFontSize {
		clamped = minFontSize
	}
	if clamped > maxFontSize {
		clamped = maxFontSize
	}
	if clamped == app.fontSize {
		return
	}
	app.fontSize = clamped
	if err := app.reloadFont(); err != nil {
		logger.Debug("reloadFont failed", "fontSize", app.fontSize, "error", err)
	}
}

func (app *App) IncreaseFontSize() {
	app.setFontSize(app.fontSize + fontSizeStep)
}

func (app *App) DecreaseFontSize() {
	app.setFontSize(app.fontSize - fontSizeStep)
}

func (app *App) ResetFontSize() {
	app.setFontSize(defaultFontSize)
}

func (app *App) CurrentScreen() Screen {
	return app.screens[app.currentScreenIndex]
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
	app.fontSize = defaultFontSize
	if err := app.reloadFont(); err != nil {
		return err
	}
	tapeScript := ""
	if app.currentFile != "" {
		tapeScript = app.openFiles[app.currentFile]
	}

	helpBytes, err := assets.ReadFile("assets/help.txt")
	if err != nil {
		return err
	}

	globalKeyMap := CreateKeyMap()
	globalKeyMap.Bind("C-g", app.Reset)
	globalKeyMap.Bind("Escape", app.Reset)
	globalKeyMap.Bind("C-z", UndoLastAction)
	globalKeyMap.Bind("C-x u", UndoLastAction)
	globalKeyMap.Bind("C-S--", UndoLastAction)
	globalKeyMap.Bind("C-q", app.Quit)
	globalKeyMap.Bind("C-S-=", app.IncreaseFontSize)
	globalKeyMap.Bind("C--", app.DecreaseFontSize)
	globalKeyMap.Bind("C-0", app.ResetFontSize)
	app.globalKeyMap = globalKeyMap

	helpScreen, err := CreateHelpScreen(app, string(helpBytes))
	if err != nil {
		return err
	}

	editScreen, err := CreateEditScreen(app, tapeScript)
	if err != nil {
		return err
	}

	fileScreen, err := CreateFileScreen(app)
	if err != nil {
		return err
	}

	app.screens = []Screen{helpScreen, editScreen, fileScreen}
	app.SelectScreen(1)

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
	app.currentScreenIndex = index
	app.currentKeyHandler = app.CurrentScreen()
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
	app.HandleKey(keyName)
}

func (app *App) HandleKey(key Key) {
	var nextHandler KeyHandler
	var handled bool
	if app.chordHandler != nil {
		nextHandler, handled = app.chordHandler.HandleKey(key)
		if handled {
			app.chordHandler = nextHandler
			return
		}
	}
	app.chordHandler = nil
	nextHandler, handled = app.currentKeyHandler.HandleKey(key)
	if handled {
		app.chordHandler = nextHandler
		return
	}
	nextHandler, handled = app.globalKeyMap.HandleKey(key)
	if handled {
		app.chordHandler = nextHandler
		return
	}
}

func (app *App) OnChar(char rune) {
	//logger.Debug("OnChar", "char", char)
	if app.chordHandler != nil {
		return
	}
	if cs, ok := app.CurrentScreen().(CharScreen); ok {
		cs.OnChar(app, char)
	}
}

func (app *App) OnFramebufferSize(width, height int) {
	logger.Debug("OnFramebufferSize", "width", width, "height", height)
}

func (app *App) BgColor() (r, g, b, a float32) {
	bg := ColorBackground
	r = float32(bg.R) / 255.0
	g = float32(bg.G) / 255.0
	b = float32(bg.B) / 255.0
	a = float32(bg.A) / 255.0
	return
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
		if err := app.vm.ParseAndEval(bytes.NewReader(editorScript), tapePath); err == nil {
			app.postEvent(func() {
				app.rTape = nil
				app.rTotalFrames = 0
				app.rDoneFrames = 0
				if evalSuccessCallback != nil {
					evalSuccessCallback()
				}
			}, false)
		}
	}()
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
	app.chordHandler = nil
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
