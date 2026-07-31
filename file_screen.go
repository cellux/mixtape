package main

import (
	"fmt"

	"github.com/atotto/clipboard"
)

// FileScreen is a simple file browser with sample playing functionality.
type FileScreen struct {
	fileBrowser *FileBrowser
	keymap      KeyMap
	app         *App

	lastPlayedPath string
	lastTape       *Tape
	lastPlayers    []*TapePlayer
	tapeDisplay    *TapeDisplay
}

func CreateFileScreen(app *App) (*FileScreen, error) {
	keymap := CreateKeyMap()
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}
	fileBrowser, err := CreateFileBrowser("", nil, FileBrowserCallbacks{})
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
	full := canonicalPath(entry.path)
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
	fs.lastPlayers = nil
	_ = fs.fileBrowser.Reset()
}

func (fs *FileScreen) Close() {}

func (fs *FileScreen) Render(app *App, ts *TileScreen) {
	pane := ts.GetPane()

	browserPane := pane
	if fs.lastTape != nil {
		var tapePane TilePane
		browserPane, tapePane = pane.SplitY(-8)
		playheadFrames := make([]int, 0, len(fs.lastPlayers))
		for _, player := range fs.lastPlayers {
			if player.IsPlaying() {
				playheadFrames = append(playheadFrames, player.GetCurrentFrame())
			}
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
	path := canonicalPath(entry.path)
	if path == fs.lastPlayedPath && fs.lastTape != nil {
		fs.playTape(app, fs.lastTape)
		return
	}
	tape, err := loadSample(path)
	if err != nil {
		fs.app.SetLastError(err)
		return
	}
	fs.lastPlayedPath = path
	fs.lastTape = tape
	fs.lastPlayers = nil
	fs.playTape(app, tape)
}

func (fs *FileScreen) playTape(app *App, tape *Tape) {
	if player := app.oto.PlayTape(tape, fs); player != nil {
		fs.lastPlayers = append(fs.lastPlayers, player)
	}
}
