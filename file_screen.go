package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/atotto/clipboard"
)

type fileEntry struct {
	name     string
	path     string
	size     int64
	mode     os.FileMode
	isDir    bool
	typeRune rune
}

func (fe fileEntry) GetUniqueId() any {
	return fe.path
}

func (fe fileEntry) Format() string {
	name := fe.name
	if fe.isDir {
		name += "/"
	}
	sizeText := ""
	if fe.mode.IsRegular() {
		sizeText = fmt.Sprintf("%d", fe.size)
	}
	return fmt.Sprintf("%c %-20s %s", fe.typeRune, name, sizeText)
}

// FileScreen is a simple file browser.
type FileScreen struct {
	dir         string
	entries     []fileEntry
	listDisplay *ListDisplay
	keymap      KeyMap
	lastErr     error

	lastPlayedPath string
	lastTape       *Tape
	tapeDisplay    *TapeDisplay
}

func CreateFileScreen(app *App, parent KeyMap) (*FileScreen, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	keymap := CreateKeyMap(parent)
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}
	listDisplay := CreateListDisplay()
	fs := &FileScreen{
		dir:         cwd,
		keymap:      keymap,
		listDisplay: listDisplay,
		tapeDisplay: tapeDisplay,
	}

	keymap.Bind("Up", func() { listDisplay.MoveBy(-1) })
	keymap.Bind("Down", func() { listDisplay.MoveBy(1) })
	keymap.Bind("Home", func() { listDisplay.MoveTo(0) })
	keymap.Bind("End", func() { listDisplay.MoveTo(len(listDisplay.GetFilteredEntries()) - 1) })
	keymap.Bind("PageUp", func() { listDisplay.MoveBy(-listDisplay.PageSize()) })
	keymap.Bind("PageDown", func() { listDisplay.MoveBy(listDisplay.PageSize()) })
	keymap.Bind("Enter", func() { fs.enter() })
	keymap.Bind("Backspace", func() { fs.handleBackspace() })
	keymap.Bind("M-w", func() { fs.copyPath() })
	keymap.Bind("C-p", func() { fs.playSelected(app) })

	if err := fs.reload(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *FileScreen) canonicalPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return canonical
}

func (fs *FileScreen) reload() error {
	prevSelection := fs.listDisplay.SelectedEntry()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		fs.entries = nil
		fs.listDisplay.SetEntries(nil)
		fs.lastErr = err
		return err
	}
	fs.lastErr = nil
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})

	fs.lastPlayedPath = ""
	fs.lastTape = nil

	var result []fileEntry
	if parent := filepath.Dir(fs.dir); parent != fs.dir {
		parentClean := filepath.Clean(parent)
		result = append(result, fileEntry{
			name:     "..",
			path:     parentClean,
			size:     0,
			mode:     os.ModeDir,
			isDir:    true,
			typeRune: 'd',
		})
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(fs.dir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		mode := info.Mode()
		isDir := entry.IsDir()
		if mode&os.ModeSymlink != 0 {
			if targetInfo, err := os.Stat(path); err == nil {
				if targetInfo.IsDir() {
					isDir = true
				}
			}
		}
		typeRune := '-'
		switch {
		case mode&os.ModeDir != 0:
			typeRune = 'd'
		case mode&os.ModeSymlink != 0:
			typeRune = 'l'
		}
		result = append(result, fileEntry{
			name:     name,
			path:     path,
			size:     info.Size(),
			mode:     mode,
			isDir:    isDir,
			typeRune: typeRune,
		})
	}

	fs.entries = result
	fs.listDisplay.SetEntries(entriesToList(result))
	if prevSelection != nil {
		fs.listDisplay.SelectEntry(prevSelection)
	}
	return nil
}

func entriesToList(entries []fileEntry) []ListEntry {
	res := make([]ListEntry, len(entries))
	for i := range entries {
		res[i] = entries[i]
	}
	return res
}

func (fs *FileScreen) handleBackspace() {
	if fs.listDisplay.searchText != "" {
		runes := []rune(fs.listDisplay.searchText)
		if len(runes) > 0 {
			fs.listDisplay.searchText = string(runes[:len(runes)-1])
			fs.listDisplay.SelectFiltered(0)
		}
		return
	}
	fs.goParent()
}

func (fs *FileScreen) goParent() {
	parent := filepath.Dir(fs.dir)
	if parent == fs.dir {
		return
	}
	fs.dir = parent
	fs.listDisplay.Reset()
	fs.lastPlayedPath = ""
	fs.lastTape = nil
	_ = fs.reload()
}

func (fs *FileScreen) enter() {
	if fs.listDisplay.SelectedEntry() == nil {
		return
	}
	entry := fs.listDisplay.SelectedEntry().(fileEntry)
	if !entry.isDir {
		return
	}
	fs.dir = entry.path
	fs.listDisplay.Reset()
	_ = fs.reload()
}

func (fs *FileScreen) copyPath() {
	if fs.listDisplay.SelectedEntry() == nil {
		return
	}
	entry := fs.listDisplay.SelectedEntry().(fileEntry)
	full := fs.canonicalPath(entry.path)
	_ = clipboard.WriteAll(fmt.Sprintf("=\"%s\" load=", full))
}

func (fs *FileScreen) Keymap() KeyMap {
	return fs.keymap
}

func (fs *FileScreen) Reset() {
	fs.lastErr = nil
	fs.listDisplay.searchText = ""
	fs.lastPlayedPath = ""
	fs.lastTape = nil
	_ = fs.reload()
}

func (fs *FileScreen) Close() {}

func (fs *FileScreen) Render(app *App, ts *TileScreen) {
	pane := ts.GetPane()
	header, bodyPane := pane.SplitY(1)
	header.DrawString(0, 0, fs.dir)
	if fs.listDisplay.searchText != "" {
		header.WithFgBg(ColorBlack, ColorWhite, func() {
			header.DrawString(len(fs.dir)+1, 0, fmt.Sprintf("[%s]", fs.listDisplay.searchText))
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

	fs.listDisplay.lastHeight = listPane.Height()
	fs.listDisplay.Render(&listPane)

	if fs.lastErr != nil {
		statusPane.WithFgBg(ColorWhite, ColorRed, func() {
			statusPane.DrawString(0, 0, fs.lastErr.Error())
		})
	}
}

func (fs *FileScreen) OnChar(app *App, char rune) {
	if char == 0 || char < 32 {
		return
	}
	fs.listDisplay.searchText += string(char)
	fs.listDisplay.SelectFiltered(0)
}

func (fs *FileScreen) playSelected(app *App) {
	filtered := fs.listDisplay.GetFilteredEntries()
	if len(filtered) == 0 {
		return
	}
	entry := filtered[fs.listDisplay.GetFilteredSelectionIndex()].(fileEntry)
	if entry.isDir {
		return
	}
	path := fs.canonicalPath(entry.path)
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
