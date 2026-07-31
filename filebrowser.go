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

type FileBrowserCallbacks struct {
	onExit   func()
	onSelect func(FileEntry)
}

type FileBrowser struct {
	dir     string
	entries []FileEntry
	*ListBrowser
	filter    FileFilter
	callbacks FileBrowserCallbacks
}

func CreateFileBrowser(startDir string, filter FileFilter, callbacks FileBrowserCallbacks) (*FileBrowser, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		startDir = cwd
	}
	fb := &FileBrowser{
		dir:       startDir,
		filter:    filter,
		callbacks: callbacks,
	}
	fb.ListBrowser = CreateListBrowser(fb.Directory, ListBrowserCallbacks{
		onEnter:                  fb.handleEnter,
		onExit:                   fb.Exit,
		onBackspaceWithoutFilter: func() { _, _ = fb.GoParent() },
	})
	if err := fb.Reload(); err != nil {
		return nil, err
	}
	return fb, nil
}

func (fb *FileBrowser) Directory() string {
	return fb.dir
}

func (fb *FileBrowser) SelectedEntry() *FileEntry {
	entry := fb.ListBrowser.SelectedEntry()
	if entry == nil {
		return nil
	}
	fe := entry.(FileEntry)
	return &fe
}

func (fb *FileBrowser) CurrentFilteredEntry() *FileEntry {
	filtered := fb.ListBrowser.GetFilteredEntries()
	if len(filtered) == 0 {
		return nil
	}
	idx := fb.ListBrowser.GetFilteredSelectionIndex()
	fe := filtered[idx].(FileEntry)
	return &fe
}

func (fb *FileBrowser) Reload() error {
	prevSelection := fb.ListBrowser.SelectedEntry()

	entries, err := os.ReadDir(fb.dir)
	if err != nil {
		fb.entries = nil
		fb.ListBrowser.SetEntries(nil)
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
	fb.ListBrowser.SetEntries(entriesToList(result))
	if prevSelection != nil {
		fb.ListBrowser.SelectEntry(prevSelection)
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

func (fb *FileBrowser) GoParent() (bool, error) {
	parent := filepath.Dir(fb.dir)
	if parent == fb.dir {
		return false, nil
	}
	fb.dir = parent
	fb.ListBrowser.Reset()
	err := fb.Reload()
	return true, err
}

func (fb *FileBrowser) Enter() (bool, error) {
	selected := fb.SelectedEntry()
	return fb.enterSelection(selected)
}

func (fb *FileBrowser) Reset() error {
	fb.ListBrowser.Reset()
	return fb.Reload()
}

func (fb *FileBrowser) Exit() {
	if fb.callbacks.onExit != nil {
		fb.callbacks.onExit()
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
		fb.ListBrowser.Reset()
		err := fb.Reload()
		return true, err
	}
	if fb.callbacks.onSelect != nil {
		fb.callbacks.onSelect(*selected)
	}
	return false, nil
}
