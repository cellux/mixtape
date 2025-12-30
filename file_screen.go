package main

import (
	"fmt"

	"github.com/atotto/clipboard"
)

// FileScreen is a simple file browser.
type FileScreen struct {
	fileBrowser *FileBrowser
	keymap      KeyMap
	app         *App

	lastPlayedPath string
	lastTape       *Tape
	tapeDisplay    *TapeDisplay
}

func CreateFileScreen(app *App) (*FileScreen, error) {
	keymap := CreateKeyMap()
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}
	fileBrowser, err := CreateFileBrowser(app, "", nil, nil, nil)
	if err != nil {
		return nil, err
	}
	fs := &FileScreen{
		fileBrowser: fileBrowser,
		keymap:      keymap,
		tapeDisplay: tapeDisplay,
		app:         app,
	}
	keymap.Bind("M-w", func() { fs.copyPath() })
	keymap.Bind("C-p", func() { fs.playSelected(app) })
	return fs, nil
}

func (fs *FileScreen) copyPath() {
	entry := fs.fileBrowser.SelectedEntry()
	if entry == nil {
		return
	}
	full := fs.fileBrowser.CanonicalPath(entry.path)
	_ = clipboard.WriteAll(fmt.Sprintf("\"%s\" load", full))
}

func (fs *FileScreen) Keymap() KeyMap {
	return fs.keymap
}

func (fs *FileScreen) HandleKey(key Key) (nextHandler KeyHandler, handled bool) {
	nextHandler, handled = fs.keymap.HandleKey(key)
	if handled {
		return
	}
	nextHandler, handled = fs.fileBrowser.HandleKey(key)
	if handled {
		return
	}
	return nil, false
}

func (fs *FileScreen) Reset() {
	fs.lastPlayedPath = ""
	fs.lastTape = nil
	_ = fs.fileBrowser.Reset()
}

func (fs *FileScreen) Close() {}

func (fs *FileScreen) Render(app *App, ts *TileScreen) {
	pane := ts.GetPane()

	browserPane := pane
	if fs.lastTape != nil {
		var tapePane TilePane
		browserPane, tapePane = pane.SplitY(-8)
		playheadFrames := []int{}
		for _, tp := range app.oto.GetTapePlayers(fs) {
			playheadFrames = append(playheadFrames, tp.GetCurrentFrame())
		}
		fs.tapeDisplay.Render(fs.lastTape, tapePane.GetPixelRect(), fs.lastTape.nframes, 0, playheadFrames)
	}

	fs.fileBrowser.Render(browserPane)
}

func (fs *FileScreen) OnChar(app *App, char rune) {
	fs.fileBrowser.OnChar(char)
}

func (fs *FileScreen) playSelected(app *App) {
	entry := fs.fileBrowser.CurrentFilteredEntry()
	if entry == nil || entry.isDir {
		return
	}
	path := fs.fileBrowser.CanonicalPath(entry.path)
	if path == fs.lastPlayedPath && fs.lastTape != nil {
		app.oto.PlayTape(fs.lastTape, fs)
		return
	}
	tape, err := loadSample(path)
	if err != nil {
		fs.app.SetLastError(err)
		return
	}
	fs.lastPlayedPath = path
	fs.lastTape = tape
	app.oto.PlayTape(tape, fs)
}
