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

// FileScreen is a simple file browser.
type FileScreen struct {
	dir        string
	entries    []fileEntry
	index      int
	top        int
	keymap     KeyMap
	lastHeight int
	lastErr    error
	searchText string
}

func CreateFileScreen(app *App, parent KeyMap) (*FileScreen, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	keymap := CreateKeyMap(parent)
	fs := &FileScreen{
		dir:    cwd,
		keymap: keymap,
	}

	keymap.Bind("Up", func() { fs.moveBy(-1) })
	keymap.Bind("Down", func() { fs.moveBy(1) })
	keymap.Bind("Home", func() { fs.moveTo(0) })
	keymap.Bind("End", func() { fs.moveTo(len(fs.filteredEntries()) - 1) })
	keymap.Bind("PageUp", func() { fs.moveBy(-fs.pageSize()) })
	keymap.Bind("PageDown", func() { fs.moveBy(fs.pageSize()) })
	keymap.Bind("Enter", func() { fs.enter() })
	keymap.Bind("Backspace", func() { fs.handleBackspace() })
	keymap.Bind("M-w", func() { fs.copyPath() })

	if err := fs.reload(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *FileScreen) pageSize() int {
	if fs.lastHeight > 0 {
		return fs.lastHeight
	}
	return 1
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
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		fs.entries = nil
		fs.index = 0
		fs.top = 0
		fs.lastErr = err
		return err
	}
	fs.lastErr = nil
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})

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
	if fs.index >= len(fs.entries) {
		fs.index = len(fs.entries) - 1
	}
	if fs.index < 0 {
		fs.index = 0
	}
	fs.ensureVisible()
	return nil
}

func (fs *FileScreen) filteredEntries() []fileEntry {
	if fs.searchText == "" {
		return fs.entries
	}
	needle := strings.ToLower(fs.searchText)
	var out []fileEntry
	for _, e := range fs.entries {
		if strings.Contains(strings.ToLower(e.name), needle) {
			out = append(out, e)
		}
	}
	return out
}

func (fs *FileScreen) filteredSelectionIndex() int {
	filtered := fs.filteredEntries()
	if len(filtered) == 0 {
		return 0
	}
	currentPath := filtered[0].path
	if fs.index >= 0 && fs.index < len(fs.entries) {
		currentPath = fs.entries[fs.index].path
	}
	for i, e := range filtered {
		if e.path == currentPath {
			return i
		}
	}
	return 0
}

func (fs *FileScreen) selectFiltered(idx int) {
	filtered := fs.filteredEntries()
	if len(filtered) == 0 {
		fs.index = 0
		fs.top = 0
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(filtered) {
		idx = len(filtered) - 1
	}
	selected := filtered[idx]
	for i, e := range fs.entries {
		if e.path == selected.path {
			fs.index = i
			break
		}
	}
	fs.ensureVisible()
}

func (fs *FileScreen) handleBackspace() {
	if fs.searchText != "" {
		runes := []rune(fs.searchText)
		if len(runes) > 0 {
			fs.searchText = string(runes[:len(runes)-1])
			fs.selectFiltered(0)
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
	fs.index = 0
	fs.top = 0
	fs.searchText = ""
	_ = fs.reload()
}

func (fs *FileScreen) ensureVisible() {
	if fs.lastHeight <= 0 {
		return
	}
	filtered := fs.filteredEntries()
	if len(filtered) == 0 {
		fs.top = 0
		fs.index = 0
		return
	}
	selIdx := fs.filteredSelectionIndex()
	if fs.top < 0 {
		fs.top = 0
	}
	if selIdx < fs.top {
		fs.top = selIdx
	}
	if selIdx >= fs.top+fs.lastHeight {
		fs.top = selIdx - fs.lastHeight + 1
	}
	maxTop := len(filtered) - 1
	if fs.top > maxTop {
		fs.top = maxTop
		if fs.top < 0 {
			fs.top = 0
		}
	}
}

func (fs *FileScreen) moveBy(delta int) {
	selIdx := fs.filteredSelectionIndex()
	fs.selectFiltered(selIdx + delta)
}

func (fs *FileScreen) moveTo(idx int) {
	fs.selectFiltered(idx)
}

func (fs *FileScreen) enter() {
	filtered := fs.filteredEntries()
	if len(filtered) == 0 {
		return
	}
	entry := filtered[fs.filteredSelectionIndex()]
	if !entry.isDir {
		return
	}
	fs.dir = entry.path
	fs.index = 0
	fs.top = 0
	fs.searchText = ""
	_ = fs.reload()
}

func (fs *FileScreen) copyPath() {
	if len(fs.entries) == 0 {
		return
	}
	entry := fs.entries[fs.index]
	full := fs.canonicalPath(entry.path)
	_ = clipboard.WriteAll(fmt.Sprintf("=\"%s\" load=", full))
}

func (fs *FileScreen) Keymap() KeyMap {
	return fs.keymap
}

func (fs *FileScreen) Reset() {
	fs.lastErr = nil
	fs.searchText = ""
	_ = fs.reload()
}

func (fs *FileScreen) Close() {}

func (fs *FileScreen) Render(app *App, ts *TileScreen) {
	pane := ts.GetPane()
	header, listPane := pane.SplitY(1)
	header.DrawString(0, 0, fs.dir)
	if fs.searchText != "" {
		header.WithFgBg(ColorBlack, ColorWhite, func() {
			header.DrawString(len(fs.dir)+1, 0, fmt.Sprintf("[%s]", fs.searchText))
		})
	}

	fs.lastHeight = listPane.Height()
	if fs.lastHeight <= 0 {
		return
	}
	fs.ensureVisible()

	availableWidth := listPane.Width()
	if availableWidth <= 0 {
		return
	}

	filtered := fs.filteredEntries()

	maxNameWidth := 1
	maxSizeWidth := 0
	for _, e := range filtered {
		name := e.name
		if e.isDir {
			name += "/"
		}
		if l := len([]rune(name)); l > maxNameWidth {
			maxNameWidth = l
		}
		if e.mode.IsRegular() {
			if w := len(fmt.Sprintf("%d", e.size)); w > maxSizeWidth {
				maxSizeWidth = w
			}
		}
	}

	minNameWidth := 1
	nameWidth := maxNameWidth
	reserve := 1 + 1
	if maxSizeWidth > 0 {
		reserve += 1 + maxSizeWidth
	}
	maxAllowedName := availableWidth - reserve
	if maxAllowedName < minNameWidth {
		nameWidth = minNameWidth
	} else if nameWidth > maxAllowedName {
		nameWidth = maxAllowedName
	}

	row := 0
	for i := fs.top; i < len(filtered) && row < fs.lastHeight; i, row = i+1, row+1 {
		entry := filtered[i]
		name := entry.name
		if entry.isDir {
			name += "/"
		}
		if runeCount := len([]rune(name)); runeCount > nameWidth {
			nameRunes := []rune(name)
			name = string(nameRunes[:nameWidth])
		}
		sizeText := ""
		if maxSizeWidth > 0 && entry.mode.IsRegular() {
			sizeText = fmt.Sprintf("%d", entry.size)
		}
		line := fmt.Sprintf("%c %-*s %s", entry.typeRune, nameWidth, name, sizeText)
		isSelected := false
		if fs.index >= 0 && fs.index < len(fs.entries) {
			isSelected = entry.path == fs.entries[fs.index].path
		}
		if isSelected {
			listPane.WithFgBg(ColorBlack, ColorWhite, func() {
				listPane.DrawString(0, row, line)
			})
		} else {
			listPane.DrawString(0, row, line)
		}
	}

	if fs.lastErr != nil {
		listPane.WithFgBg(ColorWhite, ColorRed, func() {
			listPane.DrawString(0, fs.lastHeight-1, fs.lastErr.Error())
		})
	}
}

func (fs *FileScreen) OnChar(app *App, char rune) {
	if char == 0 || char < 32 {
		return
	}
	fs.searchText += string(char)
	fs.selectFiltered(0)
}
