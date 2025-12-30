package main

import "fmt"

// BufferEntry adapts Buffer to the ListEntry interface.
type BufferEntry struct {
	buffer *Buffer
}

func (be BufferEntry) GetUniqueId() any {
	return be.buffer
}

func (be BufferEntry) Format() string {
	path := be.buffer.Path
	if path == "" {
		path = "(scratch)"
	}
	return fmt.Sprintf("%-20s %s", be.buffer.Name, path)
}

type BufferBrowserCallbacks struct {
	onSelect func(*Buffer)
	onExit   func()
}

// BufferBrowser provides a searchable list of buffers.
type BufferBrowser struct {
	app         *App
	listDisplay *ListDisplay
	keymap      KeyMap
	callbacks   BufferBrowserCallbacks
}

func CreateBufferBrowser(app *App, callbacks BufferBrowserCallbacks) *BufferBrowser {
	bb := &BufferBrowser{
		app:         app,
		listDisplay: CreateListDisplay(),
		callbacks:   callbacks,
	}
	bb.initKeymap()
	bb.Reload()
	return bb
}

func (bb *BufferBrowser) initKeymap() {
	bb.keymap = CreateKeyMap()
	bb.keymap.Bind("Up", func() { bb.MoveBy(-1) })
	bb.keymap.Bind("Down", func() { bb.MoveBy(1) })
	bb.keymap.Bind("Home", func() { bb.MoveTo(0) })
	bb.keymap.Bind("End", func() { bb.MoveToEnd() })
	bb.keymap.Bind("PageUp", func() { bb.MoveBy(-bb.PageSize()) })
	bb.keymap.Bind("PageDown", func() { bb.MoveBy(bb.PageSize()) })
	bb.keymap.Bind("Backspace", func() { bb.HandleBackspace() })
	bb.keymap.Bind("Enter", func() { bb.handleEnter() })
	bb.keymap.Bind("Escape", func() { bb.Exit() })
	bb.keymap.Bind("C-g", func() { bb.Exit() })
}

func (bb *BufferBrowser) SearchText() string {
	return bb.listDisplay.SearchText()
}

func (bb *BufferBrowser) Reload() {
	entries := make([]ListEntry, len(bb.app.buffers))
	for i, buf := range bb.app.buffers {
		entries[i] = BufferEntry{buffer: buf}
	}
	bb.listDisplay.SetEntries(entries)
	if bb.app.currentBuffer != nil {
		_ = bb.listDisplay.SelectById(bb.app.currentBuffer)
	}
}

func (bb *BufferBrowser) MoveBy(delta int) {
	bb.listDisplay.MoveBy(delta)
}

func (bb *BufferBrowser) MoveTo(idx int) {
	bb.listDisplay.MoveTo(idx)
}

func (bb *BufferBrowser) MoveToEnd() {
	bb.MoveTo(len(bb.listDisplay.GetFilteredEntries()) - 1)
}

func (bb *BufferBrowser) PageSize() int {
	return bb.listDisplay.PageSize()
}

func (bb *BufferBrowser) CurrentFilteredEntry() *Buffer {
	filtered := bb.listDisplay.GetFilteredEntries()
	if len(filtered) == 0 {
		return nil
	}
	idx := bb.listDisplay.GetFilteredSelectionIndex()
	be := filtered[idx].(BufferEntry)
	return be.buffer
}

func (bb *BufferBrowser) Keymap() KeyMap {
	return bb.keymap
}

func (bb *BufferBrowser) HandleKey(key Key) (KeyHandler, bool) {
	return bb.keymap.HandleKey(key)
}

func (bb *BufferBrowser) OnChar(char rune) {
	bb.listDisplay.AppendSearchChar(char)
}

func (bb *BufferBrowser) HandleBackspace() {
	if bb.listDisplay.RemoveLastSearchChar() {
		return
	}
}

func (bb *BufferBrowser) Reset() {
	bb.listDisplay.Reset()
	bb.Reload()
}

func (bb *BufferBrowser) Exit() {
	if bb.callbacks.onExit != nil {
		bb.callbacks.onExit()
	}
}

func (bb *BufferBrowser) handleEnter() {
	buf := bb.CurrentFilteredEntry()
	if buf == nil {
		return
	}
	if bb.callbacks.onSelect != nil {
		bb.callbacks.onSelect(buf)
	}
}

func (bb *BufferBrowser) Render(tp TilePane) {
	height := tp.Height()
	if height <= 0 {
		return
	}

	header := tp.SubPane(0, 0, tp.Width(), 1)
	header.DrawString(0, 0, "Buffers")
	if bb.SearchText() != "" {
		header.WithFgBg(ColorWhite, ColorGreen, func() {
			header.DrawString(len("Buffers")+1, 0, fmt.Sprintf("[%s]", bb.SearchText()))
		})
	}

	listPane := tp.SubPane(0, 1, tp.Width(), height-1)
	bb.listDisplay.Render(listPane)
}
