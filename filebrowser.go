package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type FileEntry struct {
	name     string
	path     string
	size     int64
	mode     os.FileMode
	isDir    bool
	typeRune rune
}

func (fe FileEntry) GetUniqueId() any {
	return fe.path
}

func (fe FileEntry) Format() string {
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

type FileFilter func(FileEntry) bool

type FileBrowser struct {
	app         *App
	dir         string
	entries     []FileEntry
	listDisplay *ListDisplay
	filter      FileFilter
	keymap      KeyMap
	onExit      func()
	onSelect    func(FileEntry)
}

func (fb *FileBrowser) initKeymap() {
	fb.keymap = CreateKeyMap()
	fb.keymap.Bind("Up", func() { fb.MoveBy(-1) })
	fb.keymap.Bind("Down", func() { fb.MoveBy(1) })
	fb.keymap.Bind("Home", func() { fb.MoveTo(0) })
	fb.keymap.Bind("End", func() { fb.MoveToEnd() })
	fb.keymap.Bind("PageUp", func() { fb.MoveBy(-fb.PageSize()) })
	fb.keymap.Bind("PageDown", func() { fb.MoveBy(fb.PageSize()) })
	fb.keymap.Bind("Enter", func() { fb.handleEnter() })
	fb.keymap.Bind("Backspace", func() { _, _ = fb.HandleBackspace() })
	fb.keymap.Bind("Escape", func() { fb.Exit() })
	fb.keymap.Bind("C-g", func() { fb.Exit() })
}

func (fb *FileBrowser) Keymap() KeyMap {
	return fb.keymap
}

func (fb *FileBrowser) HandleKey(key Key) (KeyHandler, bool) {
	return fb.keymap.HandleKey(key)
}

func CreateFileBrowser(app *App, startDir string, filter FileFilter, onSelect func(FileEntry), onExit func()) (*FileBrowser, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		startDir = cwd
	}
	fb := &FileBrowser{
		app:         app,
		dir:         startDir,
		listDisplay: CreateListDisplay(),
		filter:      filter,
		onSelect:    onSelect,
		onExit:      onExit,
	}
	fb.initKeymap()
	if err := fb.Reload(); err != nil {
		return nil, err
	}
	return fb, nil
}

func (fb *FileBrowser) Directory() string {
	return fb.dir
}

func (fb *FileBrowser) SearchText() string {
	return fb.listDisplay.SearchText()
}

func (fb *FileBrowser) MoveBy(delta int) {
	fb.listDisplay.MoveBy(delta)
}

func (fb *FileBrowser) MoveTo(idx int) {
	fb.listDisplay.MoveTo(idx)
}

func (fb *FileBrowser) MoveToEnd() {
	fb.MoveTo(len(fb.listDisplay.GetFilteredEntries()) - 1)
}

func (fb *FileBrowser) PageSize() int {
	return fb.listDisplay.PageSize()
}

func (fb *FileBrowser) SelectedEntry() *FileEntry {
	entry := fb.listDisplay.SelectedEntry()
	if entry == nil {
		return nil
	}
	fe := entry.(FileEntry)
	return &fe
}

func (fb *FileBrowser) CurrentFilteredEntry() *FileEntry {
	filtered := fb.listDisplay.GetFilteredEntries()
	if len(filtered) == 0 {
		return nil
	}
	idx := fb.listDisplay.GetFilteredSelectionIndex()
	fe := filtered[idx].(FileEntry)
	return &fe
}

func (fb *FileBrowser) CanonicalPath(p string) string {
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

func (fb *FileBrowser) Reload() error {
	prevSelection := fb.listDisplay.SelectedEntry()

	entries, err := os.ReadDir(fb.dir)
	if err != nil {
		fb.entries = nil
		fb.listDisplay.SetEntries(nil)
		fb.app.SetLastError(err)
		return err
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})

	var result []FileEntry
	if parent := filepath.Dir(fb.dir); parent != fb.dir {
		parentClean := filepath.Clean(parent)
		result = append(result, FileEntry{
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
		path := filepath.Join(fb.dir, name)
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
		fileEntry := FileEntry{
			name:     name,
			path:     path,
			size:     info.Size(),
			mode:     mode,
			isDir:    isDir,
			typeRune: typeRune,
		}
		if fb.filter != nil && !fb.filter(fileEntry) {
			continue
		}
		result = append(result, fileEntry)
	}

	fb.entries = result
	fb.listDisplay.SetEntries(entriesToList(result))
	if prevSelection != nil {
		fb.listDisplay.SelectEntry(prevSelection)
	}
	return nil
}

func entriesToList(entries []FileEntry) []ListEntry {
	res := make([]ListEntry, len(entries))
	for i := range entries {
		res[i] = entries[i]
	}
	return res
}

func (fb *FileBrowser) HandleBackspace() (bool, error) {
	if fb.listDisplay.RemoveLastSearchChar() {
		return false, nil
	}
	return fb.GoParent()
}

func (fb *FileBrowser) GoParent() (bool, error) {
	parent := filepath.Dir(fb.dir)
	if parent == fb.dir {
		return false, nil
	}
	fb.dir = parent
	fb.listDisplay.Reset()
	err := fb.Reload()
	return true, err
}

func (fb *FileBrowser) Enter() (bool, error) {
	selected := fb.SelectedEntry()
	return fb.enterSelection(selected)
}

func (fb *FileBrowser) OnChar(char rune) {
	fb.listDisplay.AppendSearchChar(char)
}

func (fb *FileBrowser) Reset() error {
	fb.listDisplay.Reset()
	return fb.Reload()
}

func (fb *FileBrowser) Exit() {
	if fb.onExit != nil {
		fb.onExit()
	}
}

func (fb *FileBrowser) handleEnter() {
	selected := fb.CurrentFilteredEntry()
	fb.enterSelection(selected)
}

func (fb *FileBrowser) enterSelection(selected *FileEntry) (bool, error) {
	if selected == nil {
		return false, nil
	}
	if selected.isDir {
		fb.dir = selected.path
		fb.listDisplay.Reset()
		err := fb.Reload()
		return true, err
	}
	if fb.onSelect != nil {
		fb.onSelect(*selected)
	}
	return false, nil
}

func (fb *FileBrowser) Render(tp TilePane) {
	height := tp.Height()
	if height <= 0 {
		return
	}

	// Header with current directory and optional search text.
	header := tp.SubPane(0, 0, tp.Width(), 1)
	header.DrawString(0, 0, fb.Directory())
	if fb.SearchText() != "" {
		header.WithFgBg(ColorWhite, ColorGreen, func() {
			header.DrawString(len(fb.Directory())+1, 0, fmt.Sprintf("[%s]", fb.SearchText()))
		})
	}

	// List area beneath the header.
	listPane := tp.SubPane(0, 1, tp.Width(), height-1)
	fb.listDisplay.Render(listPane)
}
