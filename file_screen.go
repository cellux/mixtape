package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

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

	loadID          uint64
	loadCancel      context.CancelFunc
	loadingPath     string
	loadingStage    string
	loadingProgress float64
}

func CreateFileScreen(app *App) (*FileScreen, error) {
	keymap := CreateKeyMap()
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}
	fs := &FileScreen{
		keymap:      keymap,
		tapeDisplay: tapeDisplay,
		app:         app,
	}
	fileBrowser, err := CreateFileBrowser("", nil, FileBrowserCallbacks{
		onExit:   func() { fs.cancelPlayback(app) },
		onSelect: fs.handleFileBrowserSelection,
	})
	if err != nil {
		return nil, err
	}
	fs.fileBrowser = fileBrowser
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

func (fs *FileScreen) handleFileBrowserSelection(entry FileEntry) {
	if !strings.EqualFold(filepath.Ext(entry.name), ".tape") {
		return
	}
	editScreen, ok := fs.app.screens["edit"].(*EditScreen)
	if !ok {
		fs.app.SetLastError(fmt.Errorf("editor screen is unavailable"))
		return
	}
	editScreen.handleFileBrowserSelection(entry)
	fs.app.SelectScreen("edit")
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
	fs.cancelLoad()
	_ = fs.fileBrowser.Reset()
}

func (fs *FileScreen) Close() {}

func (fs *FileScreen) Render(app *App, ts *TileScreen) {
	pane := ts.GetPane()

	browserPane := pane
	if fs.lastTape != nil && fs.loadingPath != "" {
		var bottomPane, tapePane, statusPane TilePane
		browserPane, bottomPane = pane.SplitY(-9)
		tapePane, statusPane = bottomPane.SplitY(8)
		fs.renderTape(tapePane)
		fs.renderLoadingStatus(statusPane)
	} else if fs.lastTape != nil {
		var tapePane TilePane
		browserPane, tapePane = pane.SplitY(-8)
		fs.renderTape(tapePane)
	} else if fs.loadingPath != "" {
		var statusPane TilePane
		browserPane, statusPane = pane.SplitY(-1)
		fs.renderLoadingStatus(statusPane)
	}

	fs.fileBrowser.Render(browserPane)
}

func (fs *FileScreen) renderTape(pane TilePane) {
	playheadFrames := make([]int, 0, len(fs.lastPlayers))
	for _, player := range fs.lastPlayers {
		if player.IsPlaying() {
			playheadFrames = append(playheadFrames, player.GetCurrentFrame())
		}
	}
	fs.tapeDisplay.Render(fs.lastTape, pane.GetPixelRect(), fs.lastTape.nframes, 0, playheadFrames)
}

func (fs *FileScreen) renderLoadingStatus(pane TilePane) {
	progress := int(fs.loadingProgress * 100)
	pane.WithFgBg(ColorWhite, ColorBlue, func() {
		pane.Clear()
		pane.DrawString(0, 0, fmt.Sprintf("%s %d%%: %s", fs.loadingStage, progress, filepath.Base(fs.loadingPath)))
	})
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
	if path == fs.loadingPath {
		return
	}
	fs.cancelLoad()
	loadID := fs.loadID
	loadContext, cancel := context.WithCancel(context.Background())
	fs.loadCancel = cancel
	fs.loadingPath = path
	fs.loadingStage = "Loading"
	fs.loadingProgress = 0
	go fs.loadAndPlay(app, loadContext, path, loadID)
}

func (fs *FileScreen) playTape(app *App, tape *Tape) {
	if player := app.oto.PlayTape(tape, fs); player != nil {
		fs.lastPlayers = append(fs.lastPlayers, player)
	}
}

func (fs *FileScreen) cancelPlayback(app *App) {
	app.oto.StopTapePlayers(fs)
	fs.lastPlayers = nil
	fs.cancelLoad()
}

func (fs *FileScreen) cancelLoad() {
	if fs.loadCancel != nil {
		fs.loadCancel()
		fs.loadCancel = nil
	}
	fs.loadID++
	fs.loadingPath = ""
	fs.loadingStage = ""
	fs.loadingProgress = 0
}

func (fs *FileScreen) loadAndPlay(app *App, loadContext context.Context, path string, loadID uint64) {
	tape, err := loadSampleWithProgress(loadContext, path, func(stage string, progress float64) {
		app.postEvent(func() {
			if fs.loadID == loadID {
				fs.loadingStage = stage
				fs.loadingProgress = progress
			}
		}, true)
	})
	app.postEvent(func() {
		if fs.loadID != loadID {
			return
		}
		fs.loadCancel = nil
		fs.loadingPath = ""
		fs.loadingStage = ""
		fs.loadingProgress = 0
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			fs.app.SetLastError(err)
			return
		}
		fs.lastPlayedPath = path
		fs.lastTape = tape
		fs.lastPlayers = nil
		fs.playTape(app, tape)
	}, false)
}
