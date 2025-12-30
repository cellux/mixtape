package main

import (
	"fmt"

	"github.com/atotto/clipboard"
)

// FileScreen is a simple file browser.
type FileScreen struct {
	fileBrowser *FileBrowser
	keymap      KeyMap
	lastErr     error

	lastPlayedPath string
	lastTape       *Tape
	tapeDisplay    *TapeDisplay
}

func CreateFileScreen(app *App, parent KeyMap) (*FileScreen, error) {
	keymap := CreateKeyMap(parent)
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}
	fileBrowser, err := CreateFileBrowser("")
	if err != nil {
		return nil, err
	}
	fs := &FileScreen{
		fileBrowser: fileBrowser,
		keymap:      keymap,
		tapeDisplay: tapeDisplay,
	}

	keymap.Bind("Up", func() { fileBrowser.MoveBy(-1) })
	keymap.Bind("Down", func() { fileBrowser.MoveBy(1) })
	keymap.Bind("Home", func() { fileBrowser.MoveTo(0) })
	keymap.Bind("End", func() { fileBrowser.MoveToEnd() })
	keymap.Bind("PageUp", func() { fileBrowser.MoveBy(-fileBrowser.PageSize()) })
	keymap.Bind("PageDown", func() { fileBrowser.MoveBy(fileBrowser.PageSize()) })
	keymap.Bind("Enter", func() { fs.handleEnter() })
	keymap.Bind("Backspace", func() { fs.handleBackspace() })
	keymap.Bind("M-w", func() { fs.copyPath() })
	keymap.Bind("C-p", func() { fs.playSelected(app) })

	return fs, nil
}

func (fs *FileScreen) handleBackspace() {
	changedDir, err := fs.fileBrowser.HandleBackspace()
	if changedDir {
		fs.lastPlayedPath = ""
		fs.lastTape = nil
	}
	fs.lastErr = err
}

func (fs *FileScreen) handleEnter() {
	changedDir, err := fs.fileBrowser.Enter()
	if changedDir {
		fs.lastPlayedPath = ""
		fs.lastTape = nil
	}
	fs.lastErr = err
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

func (fs *FileScreen) Reset() {
	fs.lastErr = nil
	fs.lastPlayedPath = ""
	fs.lastTape = nil
	_ = fs.fileBrowser.Reset()
}

func (fs *FileScreen) Close() {}

func (fs *FileScreen) Render(app *App, ts *TileScreen) {
	pane := ts.GetPane()
	header, bodyPane := pane.SplitY(1)
	header.DrawString(0, 0, fs.fileBrowser.Directory())
	if fs.fileBrowser.SearchText() != "" {
		header.WithFgBg(ColorWhite, ColorGreen, func() {
			header.DrawString(len(fs.fileBrowser.Directory())+1, 0, fmt.Sprintf("[%s]", fs.fileBrowser.SearchText()))
		})
	}

	var statusPane TilePane
	if fs.lastErr != nil {
		bodyPane, statusPane = bodyPane.SplitY(-1)
	}

	listPane := bodyPane
	if fs.lastTape != nil {
		var tapePane TilePane
		listPane, tapePane = bodyPane.SplitY(-8)
		playheadFrames := []int{}
		for _, tp := range app.oto.GetTapePlayers(fs) {
			playheadFrames = append(playheadFrames, tp.GetCurrentFrame())
		}
		fs.tapeDisplay.Render(fs.lastTape, tapePane.GetPixelRect(), fs.lastTape.nframes, 0, playheadFrames)
	}

	fs.fileBrowser.listDisplay.lastHeight = listPane.Height()
	fs.fileBrowser.Render(&listPane)

	if fs.lastErr != nil {
		statusPane.WithFgBg(ColorWhite, ColorRed, func() {
			statusPane.DrawString(0, 0, fs.lastErr.Error())
		})
	}
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
		fs.lastErr = err
		return
	}
	fs.lastErr = nil
	fs.lastPlayedPath = path
	fs.lastTape = tape
	app.oto.PlayTape(tape, fs)
}
