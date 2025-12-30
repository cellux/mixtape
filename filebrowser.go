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
}

func CreateFileBrowser(app *App, startDir string, filter FileFilter) (*FileBrowser, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		startDir = cwd
	}
	fb := &FileBrowser{app: app, dir: startDir, listDisplay: CreateListDisplay(), filter: filter}
	if err := fb.Reload(); err != nil {
		return nil, err
	}
	return fb, nil
}

func (fb *FileBrowser) Directory() string {
	return fb.dir
}

func (fb *FileBrowser) SearchText() string {
	return fb.listDisplay.searchText
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
	if fb.listDisplay.searchText != "" {
		runes := []rune(fb.listDisplay.searchText)
		if len(runes) > 0 {
			fb.listDisplay.searchText = string(runes[:len(runes)-1])
			fb.listDisplay.SelectFiltered(0)
		}
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
	if selected == nil {
		return false, nil
	}
	if !selected.isDir {
		return false, nil
	}
	fb.dir = selected.path
	fb.listDisplay.Reset()
	err := fb.Reload()
	return true, err
}

func (fb *FileBrowser) OnChar(char rune) {
	if char == 0 || char < 32 {
		return
	}
	fb.listDisplay.searchText += string(char)
	fb.listDisplay.SelectFiltered(0)
}

func (fb *FileBrowser) Reset() error {
	fb.listDisplay.Reset()
	return fb.Reload()
}

func (fb *FileBrowser) Render(tp *TilePane) {
	fb.listDisplay.Render(tp)
}
